package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lx/internal/core/services"
	"lx/pkg/ui"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [query]",
	Short: "Clean build artifacts and cache",
	Long: `Remove compiled PDF files, logs, and auxiliary files.

If no argument is provided, this command clears the entire cache directory.
If a query is provided, it searches for a matching note and only cleans its artifacts.

Examples:
  lx clean           # Wipe entire cache (Fixes "stuck" builds)
  lx clean graph     # Remove artifacts for 'graph-theory' only`,
	RunE: runClean,
}

func runClean(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Case: Clean All (No arguments)
	if len(args) == 0 {
		fmt.Print(ui.StyleWarning.Render("Cleaning entire cache... "))

		// This wipes ~/.local/share/lx/cache/
		if err := appVault.CleanCache(); err != nil {
			fmt.Println(ui.FormatError("Failed"))
			return err
		}

		fmt.Println(ui.FormatSuccess("Done"))
		fmt.Println(ui.FormatMuted("All build artifacts and PDFs removed."))
		return nil
	}

	// 2. Case: Clean Specific Note
	query := args[0]

	// Search for the note to get the exact slug
	req := services.SearchRequest{Query: query}
	resp, err := listService.Search(ctx, req)
	if err != nil {
		return err
	}

	if resp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes found matching: " + query))
		return nil
	}

	// If multiple matches, we pick the first one (most relevant)
	target := resp.Notes[0]

	fmt.Printf("%s artifacts for '%s'... ", ui.StyleWarning.Render("Cleaning"), target.Slug)

	// Run latexmk -C on that specific file
	if err := latexCompiler.Clean(ctx, target.Slug); err != nil {
		fmt.Println(ui.FormatError("Failed"))
		return err
	}

	// Explicitly remove the PDF (latexmk -C should do this, but we double check)
	pdfPath := appVault.GetCachePath(target.Slug + ".pdf")
	if _, err := os.Stat(pdfPath); err == nil {
		os.Remove(pdfPath)
	}

	fmt.Println(ui.FormatSuccess("Done"))
	return nil
}
