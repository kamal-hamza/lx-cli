package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	cleanPrune bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean build artifacts and unused assets",
	Long: `Remove temporary files and unused assets.

Modes:
  lx clean           # Remove compiled artifacts (PDFs, logs) from cache
  lx clean --prune   # Scan assets/ folder and delete files not used in any note`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanPrune, "prune", "p", false, "Prune unused assets")
}

func runClean(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// Mode 1: Prune Assets
	if cleanPrune {
		return runPruneAssets()
	}

	// Mode 2: Clean Specific Note
	if len(args) > 0 {
		query := args[0]
		req := services.SearchRequest{Query: query}
		resp, err := listService.Search(ctx, req)
		if err != nil {
			return err
		}

		if resp.Total == 0 {
			return fmt.Errorf("no note found matching '%s'", query)
		}

		// Select the top match
		target := resp.Notes[0]
		fmt.Printf("Cleaning artifacts for: %s (%s)...\n", ui.StyleBold.Render(target.Title), target.Slug)

		// Use the compiler's Clean method to remove auxiliary files (.aux, .log, etc.)
		// This respects the logic defined in internal/adapters/compiler/
		if err := latexCompiler.Clean(ctx, target.Slug); err != nil {
			fmt.Println(ui.FormatError("Failed to clean artifacts: " + err.Error()))
			return err
		}

		fmt.Println(ui.FormatSuccess("Artifacts removed."))
		return nil
	}

	// Mode 3: Clean Entire Cache (Default)
	fmt.Print(ui.StyleWarning.Render("Cleaning entire cache... "))
	if err := appVault.CleanCache(); err != nil {
		fmt.Println(ui.FormatError("Failed"))
		return err
	}
	fmt.Println(ui.FormatSuccess("Done"))
	return nil
}

func runPruneAssets() error {
	ctx := getContext()
	fmt.Println(ui.FormatRocket("Scanning for unused assets..."))

	// 1. Load Index to find what IS used
	// Force a reindex in memory to be safe?
	// For speed, load disk index. Ideally user runs 'lx reindex' first.
	index, err := indexerService.LoadIndex()
	if err != nil {
		return fmt.Errorf("index not found. Run 'lx reindex' first")
	}

	usedAssets := make(map[string]bool)
	for _, note := range index.Notes {
		for _, asset := range note.Assets {
			usedAssets[asset] = true
		}
	}

	// 2. Scan Assets Directory
	files, err := os.ReadDir(appVault.AssetsPath)
	if err != nil {
		return err
	}

	var candidates []string
	var candidatesSize int64

	for _, f := range files {
		if f.IsDir() || f.Name() == ".manifest.json" {
			continue
		}

		if !usedAssets[f.Name()] {
			candidates = append(candidates, f.Name())
			info, _ := f.Info()
			candidatesSize += info.Size()
		}
	}

	if len(candidates) == 0 {
		fmt.Println(ui.FormatSuccess("No unused assets found."))
		return nil
	}

	// 3. Confirm
	fmt.Println()
	fmt.Println(ui.FormatWarning(fmt.Sprintf("Found %d unused assets (%s):",
		len(candidates), formatBytes(candidatesSize))))

	// Show first few
	limit := 5
	for i, c := range candidates {
		if i >= limit {
			fmt.Println(ui.FormatMuted(fmt.Sprintf("  ... and %d more", len(candidates)-limit)))
			break
		}
		fmt.Println("  " + ui.StyleError.Render("✗ "+c))
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(ui.StyleError.Render("Delete these files? (y/n): "))
	response, err := reader.ReadString('\n')
	if err != nil || strings.ToLower(strings.TrimSpace(response)) != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// 4. Delete
	count := 0
	for _, f := range candidates {
		path := appVault.GetAssetPath(f)
		if err := os.Remove(path); err == nil {
			assetRepo.Delete(ctx, f) // Update manifest
			count++
		}
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Pruned %d assets.", count)))
	return nil
}
