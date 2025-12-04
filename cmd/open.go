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
	openTemplate bool
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:     "open [query]",
	Aliases: []string{"o"},
	Short:   "Open a note PDF or template in the default viewer (alias: o)",
	Long: `Open a note PDF or template using fuzzy search.
If no query is provided, shows an interactive list to select from.

For notes, this opens the compiled PDF. For templates, this opens the .sty file.

Examples:
  # Open a note PDF
  lx open
  lx open graph
  lx open "chemistry lab"
  lx open calc

  # Open a template
  lx open -t
  lx open -t homework
  lx open --template "my custom"`,
	Args: cobra.MaximumNArgs(1),
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

	fmt.Println(ui.FormatInfo("Opening: " + selectedNote.Title))
	fmt.Println()

	// Get the PDF path (in cache directory)
	pdfName := strings.TrimSuffix(selectedNote.Filename, ".tex") + ".pdf"
	pdfPath := appVault.GetCachePath(pdfName)

	// Check if PDF exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		fmt.Println(ui.FormatWarning("PDF not found. The note may not have been built yet."))
		fmt.Println(ui.FormatInfo("Build it first with: lx build " + selectedNote.Slug))
		return nil
	}

	// Open PDF with system default viewer
	if err := OpenFileWithDefaultApp(pdfPath); err != nil {
		fmt.Println(ui.FormatError("Failed to open PDF: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually open: " + pdfPath))
		return err
	}

	return nil
}

func runOpenTemplate(cmd *cobra.Command, args []string) error {
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

	fmt.Println(ui.FormatInfo("Opening template: " + selectedTemplate.Name))
	fmt.Println()

	// Get the template file path
	templatePath := selectedTemplate.Path

	// Open with system default viewer
	if err := OpenFileWithDefaultApp(templatePath); err != nil {
		fmt.Println(ui.FormatError("Failed to open template: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually open: " + templatePath))
		return err
	}

	return nil
}
