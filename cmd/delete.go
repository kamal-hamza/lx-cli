package cmd

import (
	"fmt"
	"os"
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

	// 1. Select Note (Reuse existing selection logic)
	var selectedNote *domain.NoteHeader

	// ... (Same selection logic as before: Fuzzy find or Search) ...
	// [Copied from previous implementation for brevity, assuming standard selection]
	if len(args) == 0 {
		req := services.ListRequest{SortBy: "date"}
		resp, err := listService.Execute(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found"))
			return nil
		}
		idx, err := fuzzyfinder.Find(
			resp.Notes,
			func(i int) string { return resp.Notes[i].Title },
		)
		if err != nil {
			return nil
		}
		selectedNote = &resp.Notes[idx]
	} else {
		req := services.SearchRequest{Query: args[0]}
		resp, err := listService.Search(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			return fmt.Errorf("no note found matching '%s'", args[0])
		}
		selectedNote = &resp.Notes[0]
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

					for _, otherAsset := range otherEntry.Assets {
						if otherAsset == asset {
							isUsedElsewhere = true
							break
						}
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

	fmt.Print(ui.StyleError.Render("Delete note? (y/n): "))
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
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
		var assetResponse string
		fmt.Scanln(&assetResponse)

		if strings.ToLower(assetResponse) == "y" {
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
	// ... (Existing template deletion logic) ...
	return nil
}
