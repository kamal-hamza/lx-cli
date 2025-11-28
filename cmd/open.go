package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	openTemplate bool
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open [query]",
	Short: "Open a note or template in your editor",
	Long: `Open a note or template in your editor using fuzzy search.

Examples:
  # Open a note
  lx open graph
  lx open "chemistry lab"
  lx open calc

  # Open a template
  lx open -t homework
  lx open --template "my custom"`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

func init() {
	openCmd.Flags().BoolVarP(&openTemplate, "template", "t", false, "Open a template instead of a note")
}

func runOpen(cmd *cobra.Command, args []string) error {
	// Check if template flag is set
	if openTemplate {
		return runOpenTemplate(cmd, args)
	}

	// Otherwise, open a note
	return runOpenNote(cmd, args)
}

func runOpenNote(cmd *cobra.Command, args []string) error {
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

func runOpenTemplate(cmd *cobra.Command, args []string) error {
	query := args[0]
	ctx := getContext()

	// Get all templates
	templates, err := templateRepo.List(ctx)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list templates"))
		return err
	}

	if len(templates) == 0 {
		fmt.Println(ui.FormatWarning("No templates found"))
		return nil
	}

	// Filter templates matching the query
	var matches []domain.Template
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

	// If multiple matches, prompt user to select one
	var selectedTemplate *domain.Template
	if len(matches) > 1 {
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
		var selection int
		for {
			fmt.Print(ui.StyleInfo.Render("Select a template (1-" + strconv.Itoa(len(matches)) + "): "))

			_, err := fmt.Scanln(&selection)
			if err != nil {
				// Clear the buffer on input error
				var discard string
				fmt.Scanln(&discard)
				fmt.Println(ui.FormatWarning("Invalid input. Please enter a number."))
				continue
			}

			if selection < 1 || selection > len(matches) {
				fmt.Println(ui.FormatWarning(fmt.Sprintf("Please enter a number between 1 and %d.", len(matches))))
				continue
			}

			// Valid selection
			selectedTemplate = &matches[selection-1]
			break
		}
		fmt.Println()
	} else {
		selectedTemplate = &matches[0]
	}

	fmt.Println(ui.FormatInfo("Opening template: " + selectedTemplate.Name))
	fmt.Println()

	// Get the template file path
	templatePath := selectedTemplate.Path

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	// Launch editor
	editorCmd := exec.Command(editor, templatePath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		fmt.Println(ui.FormatError("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + templatePath))
		return err
	}

	return nil
}
