package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	editTemplate bool
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:     "edit [query]",
	Short:   "Edit a note or template in your editor",
	Aliases: []string{"e"},
	Long: `Edit a note or template in your editor using fuzzy search.
If no query is provided, shows an interactive list to select from.

Examples:
  # Edit a note
  lx edit
  lx edit graph
  lx edit "chemistry lab"
  lx edit calc

  # Edit a template
  lx edit -t
  lx edit -t homework
  lx edit --template "my custom"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().BoolVarP(&editTemplate, "template", "t", false, "Edit a template instead of a note")
}

func runEdit(cmd *cobra.Command, args []string) error {
	// Check if template flag is set
	if editTemplate {
		return runEditTemplate(cmd, args)
	}

	// Otherwise, edit a note
	return runEditNote(cmd, args)
}

func runEditNote(_ *cobra.Command, args []string) error {
	ctx := getContext()

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
			fmt.Println(ui.FormatInfo("Create your first note with: lx new \"My Note\""))
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
	var selectedNote *domain.NoteHeader
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

	fmt.Println(ui.FormatInfo("Editing: " + selectedNote.Title))
	fmt.Println()

	// Get the note file path
	notePath := appVault.GetNotePath(selectedNote.Filename)

	// Launch editor using consistent helper (line 1)
	if err := OpenEditorAtLine(notePath, 1); err != nil {
		fmt.Println(ui.FormatError("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + notePath))
		return err
	}

	return nil
}

func runEditTemplate(_ *cobra.Command, args []string) error {
	ctx := getContext()
	useFuzzyFinder := len(args) == 0

	// Get all templates
	templates, err := templateRepo.List(ctx)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list templates"))
		return err
	}

	if len(templates) == 0 {
		fmt.Println(ui.FormatWarning("No templates found"))
		fmt.Println(ui.FormatInfo("Create a template with: lx new template \"title\""))
		return nil
	}

	// Filter templates matching the query (if provided)
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

	// Select template
	var selectedTemplate *domain.Template
	if len(matches) == 1 {
		selectedTemplate = &matches[0]
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
		selectedTemplate = &matches[idx]
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
			selectedTemplate = &matches[selection-1]
			break
		}
		fmt.Println()
	}

	fmt.Println(ui.FormatInfo("Editing template: " + selectedTemplate.Name))
	fmt.Println()

	// Get the template file path
	templatePath := selectedTemplate.Path

	// Launch editor using consistent helper (line 1)
	if err := OpenEditorAtLine(templatePath, 1); err != nil {
		fmt.Println(ui.FormatError("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + templatePath))
		return err
	}

	return nil
}
