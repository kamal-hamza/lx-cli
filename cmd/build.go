package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	buildOpen bool
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [query]",
	Short: "Build a LaTeX note to PDF",
	Long: `Build a LaTeX note to PDF using latexmk.

The output PDF will be stored in the cache directory.

Examples:
  lx build graph
  lx build "chemistry lab"
  lx build calc --open`,
	Args: cobra.ExactArgs(1),
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().BoolVar(&buildOpen, "open", false, "Open the PDF after building")
}

func runBuild(cmd *cobra.Command, args []string) error {
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
