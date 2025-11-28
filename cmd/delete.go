package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	deleteTemplate bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [query]",
	Short: "Delete a note or template",
	Long: `Delete a note or template using fuzzy search.

Examples:
  # Delete a note
  lx delete graph
  lx delete "chemistry lab"
  lx delete calc

  # Delete a template
  lx delete -t homework
  lx delete --template "my custom"`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteTemplate, "template", "t", false, "Delete a template instead of a note")
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Check if template flag is set
	if deleteTemplate {
		return runDeleteTemplate(cmd, args)
	}

	// Otherwise, delete a note
	return runDeleteNote(cmd, args)
}

func runDeleteNote(cmd *cobra.Command, args []string) error {
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

	// Show warning and ask for confirmation
	fmt.Println(ui.FormatWarning("You are about to delete:"))
	fmt.Printf("  %s %s\n", ui.StyleBold.Render(selectedNote.Title), ui.StyleMuted.Render("("+selectedNote.Slug+")"))
	fmt.Println()

	// Confirmation prompt with retry loop
	var confirmed bool
	for {
		fmt.Print(ui.StyleError.Render("Are you sure you want to delete this note? (y/n): "))

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			// Clear the buffer on input error
			var discard string
			fmt.Scanln(&discard)
			fmt.Println(ui.FormatWarning("Invalid input. Please enter 'y' or 'n'."))
			continue
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			confirmed = true
			break
		} else if response == "n" || response == "no" {
			confirmed = false
			break
		} else {
			fmt.Println(ui.FormatWarning("Please enter 'y' for yes or 'n' for no."))
		}
	}

	if !confirmed {
		fmt.Println(ui.FormatInfo("Deletion cancelled."))
		return nil
	}

	// Delete the note
	if err := noteRepo.Delete(ctx, selectedNote.Slug); err != nil {
		fmt.Println(ui.FormatError("Failed to delete note: " + err.Error()))
		return err
	}

	fmt.Println(ui.FormatSuccess("Note deleted successfully: " + selectedNote.Title))
	return nil
}

func runDeleteTemplate(cmd *cobra.Command, args []string) error {
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

	// Show warning and ask for confirmation
	fmt.Println(ui.FormatWarning("You are about to delete template:"))
	fmt.Printf("  %s\n", ui.StyleBold.Render(selectedTemplate.Name))
	fmt.Println()

	// Confirmation prompt with retry loop
	var confirmed bool
	for {
		fmt.Print(ui.StyleError.Render("Are you sure you want to delete this template? (y/n): "))

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			// Clear the buffer on input error
			var discard string
			fmt.Scanln(&discard)
			fmt.Println(ui.FormatWarning("Invalid input. Please enter 'y' or 'n'."))
			continue
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			confirmed = true
			break
		} else if response == "n" || response == "no" {
			confirmed = false
			break
		} else {
			fmt.Println(ui.FormatWarning("Please enter 'y' for yes or 'n' for no."))
		}
	}

	if !confirmed {
		fmt.Println(ui.FormatInfo("Deletion cancelled."))
		return nil
	}

	// Delete the template file
	if err := os.Remove(selectedTemplate.Path); err != nil {
		fmt.Println(ui.FormatError("Failed to delete template: " + err.Error()))
		return err
	}

	fmt.Println(ui.FormatSuccess("Template deleted successfully: " + selectedTemplate.Name))
	return nil
}
