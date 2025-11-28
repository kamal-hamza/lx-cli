package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open [query]",
	Short: "Open a note in your editor",
	Long: `Open a note in your editor using fuzzy search.

Examples:
  lx open graph
  lx open "chemistry lab"
  lx open calc`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Search for notes matching the query
	req := services.SearchRequest{
		Query: query,
	}

	ctx := getContext()
	resp, err := listService.Search(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to search notes"))
		return err
	}

	// Handle no results
	if resp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes found matching: " + query))
		return nil
	}

	// If multiple matches, prompt user to select one
	var selectedNote *domain.NoteHeader
	if resp.Total > 1 {
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
		var selection int
		for {
			fmt.Print(ui.StyleInfo.Render("Select a note (1-" + strconv.Itoa(resp.Total) + "): "))

			_, err := fmt.Scanln(&selection)
			if err != nil {
				// Clear the buffer on input error
				var discard string
				fmt.Scanln(&discard)
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
	} else {
		selectedNote = &resp.Notes[0]
	}

	fmt.Println(ui.FormatInfo("Opening: " + selectedNote.Title))
	fmt.Println()

	// Get the note file path
	notePath := appVault.GetNotePath(selectedNote.Filename)

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	// Launch editor
	editorCmd := exec.Command(editor, notePath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		fmt.Println(ui.FormatError("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + notePath))
		return err
	}

	return nil
}
