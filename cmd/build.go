package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	buildOpen     bool
	buildTemplate bool
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [query]",
	Short: "Build a LaTeX note to PDF or test a template",
	Long: `Build a LaTeX note to PDF using latexmk, or test build a template.

The output PDF will be stored in the cache directory.

Examples:
  # Build a note
  lx build graph
  lx build "chemistry lab"
  lx build calc --open

  # Test build a template
  lx build -t homework
  lx build --template "my custom"`,
	Args: cobra.ExactArgs(1),
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
	query := args[0]

	// Search for the note
	searchReq := services.SearchRequest{
		Query: query,
	}

	ctx := getContext()
	searchResp, err := listService.Search(ctx, searchReq)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to search notes"))
		return err
	}

	// Handle no results
	if searchResp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes found matching: " + query))
		return nil
	}

	// Select the first match
	selectedNote := searchResp.Notes[0]

	// Show what we're building
	if searchResp.Total > 1 {
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Found %d matches, building first:", searchResp.Total)))
		fmt.Println(ui.StyleAccent.Render("→ " + selectedNote.Title + " (" + selectedNote.Slug + ")"))
		fmt.Println()
	} else {
		fmt.Println(ui.FormatInfo("Building: " + selectedNote.Title))
		fmt.Println()
	}

	// Build the note
	buildReq := services.BuildRequest{
		Slug: selectedNote.Slug,
	}

	fmt.Println(ui.FormatRocket("Compiling LaTeX..."))
	buildResp, err := buildService.Execute(ctx, buildReq)
	if err != nil {
		fmt.Println(ui.FormatError("Build failed"))
		fmt.Println()
		fmt.Println(ui.FormatMuted("Error details:"))
		fmt.Println(err.Error())
		return err
	}

	// Success
	fmt.Println(ui.FormatSuccess("Build completed successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Output", buildResp.OutputPath))
	fmt.Println()

	// Open PDF if requested
	if buildOpen {
		fmt.Println(ui.FormatInfo("Opening PDF..."))
		if err := openFile(buildResp.OutputPath); err != nil {
			fmt.Println(ui.FormatWarning("Failed to open PDF: " + err.Error()))
			fmt.Println(ui.FormatInfo("You can manually open: " + buildResp.OutputPath))
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
			fmt.Print(ui.StyleInfo.Render("Select a template (1-" + fmt.Sprintf("%d", len(matches)) + "): "))

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

	fmt.Println(ui.FormatInfo("Testing template: " + selectedTemplate.Name))
	fmt.Println()

	// Create a minimal test document
	testDoc := fmt.Sprintf(`\documentclass{article}
\usepackage{%s}
\begin{document}
Test document for template validation.
\end{document}
`, selectedTemplate.Name)

	// Create a temporary file
	testFile := appVault.GetCachePath("template-test.tex")
	if err := os.WriteFile(testFile, []byte(testDoc), 0644); err != nil {
		fmt.Println(ui.FormatError("Failed to create test file: " + err.Error()))
		return err
	}
	defer os.Remove(testFile)

	// Compile the test document
	latexArgs := []string{
		"-pdf",
		"-output-directory=" + appVault.CachePath,
		"-interaction=nonstopmode",
		"-file-line-error",
		testFile,
	}

	cmdExec := exec.Command("latexmk", latexArgs...)
	cmdExec.Dir = appVault.CachePath

	// Set environment with TEXINPUTS
	cmdEnv := os.Environ()
	texinputs := appVault.GetTexInputsEnv()
	cmdEnv = append(cmdEnv, "TEXINPUTS="+texinputs)
	cmdExec.Env = cmdEnv

	fmt.Println(ui.FormatRocket("Compiling template test..."))
	// Capture output
	output, err := cmdExec.CombinedOutput()

	// Clean up auxiliary files
	cleanArgs := []string{
		"-C",
		"-output-directory=" + appVault.CachePath,
		testFile,
	}
	cleanCmd := exec.Command("latexmk", cleanArgs...)
	cleanCmd.Dir = appVault.CachePath
	cleanCmd.Run()

	// Remove PDF if created
	pdfFile := appVault.GetCachePath("template-test.pdf")
	os.Remove(pdfFile)

	if err != nil {
		fmt.Println(ui.FormatError("Template compilation failed!"))
		fmt.Println()
		fmt.Println(ui.StyleMuted.Render("Error output:"))
		fmt.Println(string(output))
		return fmt.Errorf("template has errors")
	}

	fmt.Println(ui.FormatSuccess("Template compiled successfully!"))
	fmt.Println(ui.FormatInfo("No syntax errors found in template: " + selectedTemplate.Name))

	return nil
}
