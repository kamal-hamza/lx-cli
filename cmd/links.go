package cmd

import (
	"fmt"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var linksCmd = &cobra.Command{
	Use:   "links [note]",
	Short: "Find backlinks to a note",
	Long: `Find all notes that link to or mention a specific note.

This command searches for the target note's "slug" inside all other notes.
It helps you see connections and references across your knowledge base.

Examples:
  lx links graph
  lx links "Neural Networks"`,
	RunE: runLinks,
}

func runLinks(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Identify the Target Note
	// We reuse the logic from open/edit/delete to find the note the user is talking about
	var targetNote *domain.NoteHeader

	if len(args) == 0 {
		// Interactive selection
		req := services.ListRequest{SortBy: "date"}
		resp, err := listService.Execute(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found."))
			return nil
		}

		idx, err := fuzzyfinder.Find(
			resp.Notes,
			func(i int) string {
				return resp.Notes[i].Title
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				return fmt.Sprintf("Select a note to see its backlinks\n\nTitle: %s\nSlug: %s",
					resp.Notes[i].Title, resp.Notes[i].Slug)
			}),
		)
		if err != nil {
			return nil // Cancelled
		}
		targetNote = &resp.Notes[idx]
	} else {
		// Search by query
		query := args[0]
		req := services.SearchRequest{Query: query}
		resp, err := listService.Search(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found matching: " + query))
			return nil
		}
		// Pick top result
		targetNote = &resp.Notes[0]
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Scanning for backlinks to: %s (%s)",
		ui.StyleBold.Render(targetNote.Title),
		ui.StyleMuted.Render(targetNote.Slug))))
	fmt.Println()

	// 2. Scan for the Slug
	// We use the GrepService to find occurrences of the slug string
	matches, err := grepService.Execute(ctx, targetNote.Slug)
	if err != nil {
		return err
	}

	// 3. Process and Display Results
	// We want to group matches by file
	backlinks := make(map[string][]services.GrepMatch)
	count := 0

	for _, m := range matches {
		// Exclude the note itself (self-references)
		if m.Slug == targetNote.Slug {
			continue
		}
		backlinks[m.Slug] = append(backlinks[m.Slug], m)
		count++
	}

	if len(backlinks) == 0 {
		fmt.Println(ui.FormatMuted("No backlinks found."))
		return nil
	}

	// Render
	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Found %d mentions in %d files:", count, len(backlinks))))
	fmt.Println()

	for slug, fileMatches := range backlinks {
		// Get readable title for the referencing file
		// (Optional: we could fetch the full header, but slug is often enough for speed)
		fmt.Println(ui.StyleAccent.Render("• " + slug))

		for _, m := range fileMatches {
			// Highlight the slug in the content
			content := strings.TrimSpace(m.Content)
			highlighted := strings.ReplaceAll(content, targetNote.Slug, ui.StyleWarning.Render(targetNote.Slug))

			// Print line number and snippet
			fmt.Printf("  %s %s\n",
				ui.StyleMuted.Render(fmt.Sprintf("%d:", m.LineNum)),
				highlighted)
		}
		fmt.Println()
	}

	return nil
}
