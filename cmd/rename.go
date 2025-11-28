package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var renameCmd = &cobra.Command{
	Use:   "rename [query] [new-title]",
	Short: "Rename a note by title (updates filename and metadata)",
	Long: `Rename a note by providing a new title.

This command performs two actions:
1. Updates the '% title:' metadata inside the file.
2. Renames the file on disk to match the new title's slug.

Examples:
  lx rename                         # Interactive selection -> Prompt for title
  lx rename graph                   # Search 'graph' -> Select -> Prompt for title
  lx rename graph "Advanced Graph Theory"  # Rename matching note to "Advanced..."`,
	Args: cobra.MaximumNArgs(2),
	RunE: runRename,
}

func runRename(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Determine Input Arguments
	var query string
	var newTitle string

	if len(args) > 0 {
		query = args[0]
	}
	if len(args) > 1 {
		newTitle = args[1]
	}

	// 2. Find the Note (Selection Logic)
	var selectedNote *domain.NoteHeader
	var resp *services.SearchResponse
	var err error

	// If no query (args[0]) provided, use interactive fuzzy finder on all notes
	useFuzzyFinder := len(args) == 0

	if useFuzzyFinder {
		// List all notes for fuzzy finding
		req := services.ListRequest{SortBy: "date"}
		listResp, err := listService.Execute(ctx, req)
		if err != nil {
			fmt.Println(ui.FormatError("Failed to list notes"))
			return err
		}

		if listResp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found to rename"))
			return nil
		}

		resp = &services.SearchResponse{
			Notes: listResp.Notes,
			Total: listResp.Total,
		}
	} else {
		// Search using the provided query
		req := services.SearchRequest{Query: query}
		resp, err = listService.Search(ctx, req)
		if err != nil {
			fmt.Println(ui.FormatError("Failed to search notes"))
			return err
		}

		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found matching: " + query))
			return nil
		}
	}

	// Handle the Selection
	if resp.Total == 1 && !useFuzzyFinder {
		selectedNote = &resp.Notes[0]
	} else if useFuzzyFinder {
		idx, err := fuzzyfinder.Find(
			resp.Notes,
			func(i int) string {
				return resp.Notes[i].Title
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				note := resp.Notes[i]
				preview := fmt.Sprintf("Title: %s\nSlug: %s\nDate: %s",
					note.Title,
					note.Slug,
					note.GetDisplayDate())
				if len(note.Tags) > 0 {
					preview += fmt.Sprintf("\nTags: %s", note.GetTagsString())
				}
				return preview
			}),
		)
		if err != nil {
			fmt.Println(ui.FormatInfo("Operation cancelled."))
			return nil
		}
		selectedNote = &resp.Notes[idx]
	} else {
		// Multiple matches -> Numbered List
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d matches for '%s':", resp.Total, query)))
		fmt.Println()

		for i, note := range resp.Notes {
			fmt.Printf("%s %d. %s %s\n",
				ui.StyleAccent.Render(""),
				i+1,
				ui.StyleBold.Render(note.Title),
				ui.StyleMuted.Render("("+note.Slug+")"))
		}
		fmt.Println()

		var selection int
		for {
			fmt.Print(ui.StyleInfo.Render("Select a note (1-" + fmt.Sprintf("%d", resp.Total) + "): "))
			_, err := fmt.Scanln(&selection)
			if err != nil {
				var discard string
				fmt.Scanln(&discard)
				fmt.Println(ui.FormatWarning("Invalid input."))
				continue
			}
			if selection < 1 || selection > resp.Total {
				fmt.Println(ui.FormatWarning("Invalid selection."))
				continue
			}
			selectedNote = &resp.Notes[selection-1]
			break
		}
	}

	// 3. Confirm Selection & Get New Title
	fmt.Println()
	fmt.Println(ui.FormatInfo("Selected Note:"))
	fmt.Println(ui.RenderKeyValue("Current Title", selectedNote.Title))
	fmt.Println(ui.RenderKeyValue("Current Slug", selectedNote.Slug))
	fmt.Println()

	if newTitle == "" {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print(ui.StylePrimary.Render("Enter new title: "))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			newTitle = input
			break
		}
	}

	// 4. Validate & Generate Slug
	if err := domain.ValidateTitle(newTitle); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}
	newSlug := domain.GenerateSlug(newTitle)

	// Guard: No change
	if newSlug == selectedNote.Slug {
		fmt.Println(ui.FormatWarning("New title generates the same slug. No rename needed."))
		return nil
	}

	// Guard: Destination exists
	if noteRepo.Exists(ctx, newSlug) {
		return fmt.Errorf("a note with the slug '%s' already exists", newSlug)
	}

	// 5. Update Metadata in File Content
	oldFilename := selectedNote.Filename
	oldPath := appVault.GetNotePath(oldFilename)

	contentBytes, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Replace the title line using Regex
	// Matches "% title: Old Title" and replaces with "% title: New Title"
	titleRegex := regexp.MustCompile(`(?m)^%\s*title:.*$`)
	newMetadataLine := fmt.Sprintf("%% title: %s", newTitle)

	if titleRegex.MatchString(content) {
		content = titleRegex.ReplaceAllString(content, newMetadataLine)
	} else {
		// Fallback: If for some reason the metadata line is missing, prepend it
		content = newMetadataLine + "\n" + content
	}

	// Write content back to the OLD path first
	if err := os.WriteFile(oldPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// 6. Rename the File
	var newFilename string
	if strings.Contains(oldFilename, "-") {
		parts := strings.SplitN(oldFilename, "-", 2)
		if len(parts[0]) == 8 {
			// Preserve date prefix
			datePrefix := parts[0]
			newFilename = fmt.Sprintf("%s-%s.tex", datePrefix, newSlug)
		} else {
			newFilename = domain.GenerateFilename(newSlug)
		}
	} else {
		newFilename = domain.GenerateFilename(newSlug)
	}

	newPath := appVault.GetNotePath(newFilename)
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess("Renamed successfully!"))
	fmt.Println(ui.RenderKeyValue("New Title", newTitle))
	fmt.Println(ui.RenderKeyValue("New Slug", newSlug))
	fmt.Println(ui.RenderKeyValue("File", newFilename))

	return nil
}
