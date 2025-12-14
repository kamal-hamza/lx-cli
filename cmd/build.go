package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	buildOpen     bool
	buildTemplate bool
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:     "build [query]",
	Short:   "Build a LaTeX note to PDF or test a template",
	Aliases: []string{"b"},
	Long: `Build a LaTeX note to PDF using latexmk, or test build a template.
If no query is provided, shows an interactive list to select from.

The output PDF will be stored in the cache directory.

Examples:
  # Build a note
  lx build
  lx build graph
  lx build "chemistry lab"
  lx build calc --open

  # Test build a template
  lx build -t
  lx build -t homework
  lx build --template "my custom"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().BoolVar(&buildOpen, "open", false, "Open the PDF after building")
	buildCmd.Flags().BoolVarP(&buildTemplate, "template", "t", false, "Build a template instead of a note")
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Check if template flag is set
	if buildTemplate {
		return runBuildTemplate(cmd, args)
	}

	// Otherwise, build a note
	return runBuildNote(cmd, args)
}

func runBuildNote(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	var searchResp *services.SearchResponse
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

		searchResp = &services.SearchResponse{
			Notes: listResp.Notes,
			Total: listResp.Total,
		}
	} else {
		query := args[0]

		// Search for the note
		searchReq := services.SearchRequest{
			Query: query,
		}

		searchResp, err = listService.Search(ctx, searchReq)
		if err != nil {
			fmt.Println(ui.FormatError("Failed to search notes"))
			return err
		}

		// Handle no results
		if searchResp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found matching: " + query))
			return nil
		}
	}

	// Select note
	var selectedNote *domain.NoteHeader
	if searchResp.Total == 1 {
		selectedNote = &searchResp.Notes[0]
	} else if useFuzzyFinder {
		// Use fuzzy finder when no query was provided
		idx, err := fuzzyfinder.Find(
			searchResp.Notes,
			func(i int) string {
				return searchResp.Notes[i].Title
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				note := searchResp.Notes[i]
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
		selectedNote = &searchResp.Notes[idx]
	} else {
		// Use numbered list when query was provided
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d matches:", searchResp.Total)))
		fmt.Println()

		// Display numbered list of notes
		for i, note := range searchResp.Notes {
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
			fmt.Print(ui.StyleInfo.Render("Select a note (1-" + fmt.Sprintf("%d", searchResp.Total) + "): "))

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

			if selection < 1 || selection > searchResp.Total {
				fmt.Println(ui.FormatWarning(fmt.Sprintf("Please enter a number between 1 and %d.", searchResp.Total)))
				continue
			}

			// Valid selection
			selectedNote = &searchResp.Notes[selection-1]
			break
		}
		fmt.Println()
	}

	fmt.Println(ui.FormatInfo("Building: " + selectedNote.Title))
	fmt.Println()

	// Build the note using BuildService
	// This will now use the Preprocessor -> Compiler pipeline
	buildReq := services.BuildRequest{
		Slug: selectedNote.Slug,
	}

	fmt.Println(ui.FormatRocket("Compiling LaTeX..."))

	// Get detailed build results
	buildDetails, err := buildService.ExecuteWithDetails(ctx, buildReq)
	if err != nil {
		fmt.Println(ui.FormatError("Build failed"))
		fmt.Println()
		if buildDetails != nil && buildDetails.Parsed != nil {
			fmt.Println(buildDetails.Parsed.FormatIssues())
		} else {
			fmt.Println(ui.FormatMuted("Error details:"))
			fmt.Println(err.Error())
		}
		return err
	}

	// Success - show summary
	if buildDetails.Parsed != nil {
		fmt.Println(buildDetails.Parsed.GetSummary())

		// Show warnings if any
		if len(buildDetails.Parsed.Warnings) > 0 {
			fmt.Println(ui.FormatMuted(fmt.Sprintf("\n⚠️  %d warning(s) (LaTeX warnings can usually be ignored)", len(buildDetails.Parsed.Warnings))))
		}
	} else {
		fmt.Println(ui.FormatSuccess("Build completed successfully!"))
	}

	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Output", buildDetails.OutputPath))
	fmt.Println()

	// Open PDF if requested
	if buildOpen {
		fmt.Println(ui.FormatInfo("Opening PDF..."))
		if err := OpenFile(buildDetails.OutputPath, appConfig.PDFViewer); err != nil {
			fmt.Println(ui.FormatWarning("Failed to open PDF: " + err.Error()))
			fmt.Println(ui.FormatInfo("You can manually open: " + buildDetails.OutputPath))
		}
	}

	return nil
}

// openFile opens a file with the system's default application
func openFile(path string) error {
	var cmd *exec.Cmd

	// Platform-specific open commands
	switch os.Getenv("GOOS") {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		// Try xdg-open on Linux
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Run()
}

func runBuildTemplate(cmd *cobra.Command, args []string) error {
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

	fmt.Println(ui.FormatInfo("Testing template: " + selectedTemplate.Name))
	fmt.Println()

	// Create a minimal test document
	testDoc := fmt.Sprintf(`\documentclass{article}
\usepackage{%s}
\begin{document}
Test document for template validation.
\end{document}
`, selectedTemplate.Name)

	// Create a temporary file in the cache
	testFile := appVault.GetCachePath("template-test.tex")
	if err := os.WriteFile(testFile, []byte(testDoc), 0644); err != nil {
		fmt.Println(ui.FormatError("Failed to create test file: " + err.Error()))
		return err
	}
	defer os.Remove(testFile)

	fmt.Println(ui.FormatRocket("Compiling template test..."))

	// Use the compiler with detailed output
	result := latexCompiler.CompileWithOutput(ctx, testFile, nil)

	if !result.Success {
		fmt.Println(ui.FormatError("Template compilation failed!"))
		fmt.Println()
		if result.Parsed != nil {
			fmt.Println(result.Parsed.FormatIssues())
		} else {
			fmt.Println(ui.StyleMuted.Render("Error details:"))
			fmt.Println(result.Output)
		}
		return fmt.Errorf("template has errors")
	}

	// Clean up the PDF artifact
	pdfFile := appVault.GetCachePath("template-test.pdf")
	os.Remove(pdfFile)

	// Show results
	if result.Parsed != nil {
		fmt.Println(result.Parsed.GetSummary())
		if len(result.Parsed.Warnings) > 0 {
			fmt.Println(ui.FormatMuted(fmt.Sprintf("(%d warning(s) - this is normal)", len(result.Parsed.Warnings))))
		}
	} else {
		fmt.Println(ui.FormatSuccess("Template compiled successfully!"))
	}
	fmt.Println(ui.FormatInfo("Template is valid: " + selectedTemplate.Name))

	return nil
}
