package cmd

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	deleteTemplate bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete [query]",
	Short: "Delete a note or template (and orphaned assets)",
	Long: `Delete a note or template.

If deleting a note, it checks if any attached assets (images/PDFs)
become "orphaned" (not used by any other note) and offers to delete them.

Examples:
  lx delete graph
  lx delete -t homework`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteTemplate, "template", "t", false, "Delete a template instead of a note")
}

func runDelete(cmd *cobra.Command, args []string) error {
	if deleteTemplate {
		return runDeleteTemplate(cmd, args)
	}
	return runDeleteNote(cmd, args)
}

func runDeleteNote(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Select Note
	var selectedNote *domain.NoteHeader
	var resp *services.SearchResponse
	var err error
	useFuzzyFinder := len(args) == 0

	// If no query provided, get all notes for interactive selection
	if useFuzzyFinder {
		req := services.ListRequest{
			SortBy: "date",
		}
		listResp, err := listService.Execute(ctx, req)
		if err != nil {
			fmt.Println(ui.FormatError("Failed to list notes"))
			return err
		}

		if listResp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found"))
			return nil
		}

		resp = &services.SearchResponse{
			Notes: listResp.Notes,
			Total: listResp.Total,
		}
	} else {
		query := args[0]

		// Search for notes matching the query
		req := services.SearchRequest{
			Query: query,
		}

		resp, err = listService.Search(ctx, req)
		if err != nil {
			fmt.Println(ui.FormatError("Failed to search notes"))
			return err
		}

		// Handle no results
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found matching: " + query))
			return nil
		}
	}

	// Select note
	if resp.Total == 1 {
		selectedNote = &resp.Notes[0]
	} else if useFuzzyFinder {
		// Use fuzzy finder when no query was provided
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
			// User cancelled (Ctrl+C or ESC)
			fmt.Println(ui.FormatInfo("Operation cancelled."))
			return nil
		}
		selectedNote = &resp.Notes[idx]
	} else {
		// Use numbered list when query was provided
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d matches:", resp.Total)))
		fmt.Println()

		// Display numbered list of notes
		for i, note := range resp.Notes {
			fmt.Printf("%s %d. %s %s\n",
				ui.StyleAccent.Render(""),
				i+1,
				ui.StyleBold.Render(note.Title),
				ui.StyleMuted.Render("("+note.Slug+")"))
		}
		fmt.Println()

		// Prompt for selection with retry loop
		reader := bufio.NewReader(os.Stdin)
		var selection int
		for {
			fmt.Print(ui.StyleInfo.Render("Select a note (1-" + fmt.Sprintf("%d", resp.Total) + "): "))

			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input. Please enter a number."))
				continue
			}

			selection, err = strconv.Atoi(strings.TrimSpace(input))
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input. Please enter a number."))
				continue
			}

			if selection < 1 || selection > resp.Total {
				fmt.Println(ui.FormatWarning(fmt.Sprintf("Please enter a number between 1 and %d.", resp.Total)))
				continue
			}

			// Valid selection
			selectedNote = &resp.Notes[selection-1]
			break
		}
		fmt.Println()
	}

	// 2. Identify Orphaned Assets
	// We need the index to know what assets this note uses, and if anyone else uses them
	index, _ := indexerService.LoadIndex() // Ignore error, best effort

	var orphans []string
	if index != nil {
		if entry, exists := index.GetNote(selectedNote.Slug); exists {
			// Check each asset used by this note
			for _, asset := range entry.Assets {
				isUsedElsewhere := false

				// Scan all other notes
				for otherSlug, otherEntry := range index.Notes {
					if otherSlug == selectedNote.Slug {
						continue
					}

					if slices.Contains(otherEntry.Assets, asset) {
						isUsedElsewhere = true
					}
					if isUsedElsewhere {
						break
					}
				}

				if !isUsedElsewhere {
					orphans = append(orphans, asset)
				}
			}
		}
	}

	// 3. Confirmation
	fmt.Println(ui.FormatWarning("You are about to delete:"))
	fmt.Printf("  %s %s\n", ui.StyleBold.Render(selectedNote.Title), ui.StyleMuted.Render("("+selectedNote.Slug+")"))

	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println(ui.FormatInfo("The following assets will be ORPHANED (unused):"))
		for _, o := range orphans {
			fmt.Printf("  • %s\n", o)
		}
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(ui.StyleError.Render("Delete note? (y/n): "))
	response, err := reader.ReadString('\n')
	if err != nil || strings.ToLower(strings.TrimSpace(response)) != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// 4. Delete Note
	if err := noteRepo.Delete(ctx, selectedNote.Slug); err != nil {
		return err
	}
	fmt.Println(ui.FormatSuccess("Note deleted."))

	// 5. Delete Orphans (Optional)
	if len(orphans) > 0 {
		fmt.Print(ui.StyleWarning.Render(fmt.Sprintf("Delete %d orphaned assets? (y/n): ", len(orphans))))
		assetResponse, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(ui.FormatMuted("Assets kept."))
			return nil
		}

		if strings.ToLower(strings.TrimSpace(assetResponse)) == "y" {
			count := 0
			for _, filename := range orphans {
				// Delete from disk
				path := appVault.GetAssetPath(filename)
				if err := os.Remove(path); err == nil {
					// Delete from manifest
					assetRepo.Delete(ctx, filename)
					count++
				}
			}
			fmt.Println(ui.FormatSuccess(fmt.Sprintf("Cleaned up %d assets.", count)))
		} else {
			fmt.Println(ui.FormatMuted("Assets kept."))
		}
	}

	return nil
}

func runDeleteTemplate(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	useFuzzyFinder := len(args) == 0

	// 1. Fetch Templates
	templates, err := templateRepo.List(ctx)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		fmt.Println(ui.FormatWarning("No templates found."))
		return nil
	}

	// 2. Filter templates matching the query (if provided)
	var matches []domain.Template
	if useFuzzyFinder {
		// No query - show all templates
		matches = templates
	} else {
		query := args[0]
		queryLower := strings.ToLower(query)
		for _, template := range templates {
			if strings.Contains(strings.ToLower(template.Name), queryLower) {
				matches = append(matches, template)
			}
		}

		if len(matches) == 0 {
			fmt.Println(ui.FormatWarning("No templates found matching: " + query))
			return nil
		}
	}

	// 3. Select template
	var selected *domain.Template
	if len(matches) == 1 {
		selected = &matches[0]
	} else if useFuzzyFinder {
		// Use fuzzy finder when no query was provided
		idx, err := fuzzyfinder.Find(
			matches,
			func(i int) string {
				return matches[i].Name
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				template := matches[i]
				return fmt.Sprintf("Name: %s\nPath: %s", template.Name, template.Path)
			}),
		)
		if err != nil {
			// User cancelled (Ctrl+C or ESC)
			fmt.Println(ui.FormatInfo("Operation cancelled."))
			return nil
		}
		selected = &matches[idx]
	} else {
		// Use numbered list when query was provided
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d matches:", len(matches))))
		fmt.Println()

		// Display numbered list of templates
		for i, template := range matches {
			fmt.Printf("%s %d. %s\n",
				ui.StyleAccent.Render(""),
				i+1,
				ui.StyleBold.Render(template.Name))
		}
		fmt.Println()

		// Prompt for selection with retry loop
		reader := bufio.NewReader(os.Stdin)
		var selection int
		for {
			fmt.Print(ui.StyleInfo.Render("Select a template (1-" + fmt.Sprintf("%d", len(matches)) + "): "))

			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input. Please enter a number."))
				continue
			}

			selection, err = strconv.Atoi(strings.TrimSpace(input))
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input. Please enter a number."))
				continue
			}

			if selection < 1 || selection > len(matches) {
				fmt.Println(ui.FormatWarning(fmt.Sprintf("Please enter a number between 1 and %d.", len(matches))))
				continue
			}

			// Valid selection
			selected = &matches[selection-1]
			break
		}
		fmt.Println()
	}

	// 4. Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(ui.StyleWarning.Render("Delete template '%s'? (y/n): "), selected.Name)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Cancelled.")
		return nil
	}

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// 5. Delete File
	if err := os.Remove(selected.Path); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	fmt.Println(ui.FormatSuccess("Template deleted."))
	return nil
}
