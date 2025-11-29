package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/assets"
	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	exportAllFormat string
	exportAllOutput string
	exportAllJobs   int
)

var exportAllCmd = &cobra.Command{
	Use:   "export-all",
	Short: "Export the entire vault to a specific format",
	Long: `Bulk export all notes in the vault.

Uses concurrent workers to process notes in parallel.
Useful for backups, static site generation, or sharing your vault.

Examples:
  lx export-all -f markdown -o ./dist
  lx export-all --format html --jobs 8`,
	RunE: runExportAll,
}

func init() {
	exportAllCmd.Flags().StringVarP(&exportAllFormat, "format", "f", "markdown", "Output format (markdown, html, docx)")
	exportAllCmd.Flags().StringVarP(&exportAllOutput, "output", "o", "", "Output directory (default: vault/exports/<format>)")
	exportAllCmd.Flags().IntVarP(&exportAllJobs, "jobs", "j", 4, "Number of concurrent workers")
}

func runExportAll(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Dependencies Check
	if err := checkAndInstallPandoc(); err != nil {
		return err
	}

	profile, ok := exportProfiles[exportAllFormat]
	if !ok {
		return fmt.Errorf("unsupported format: %s", exportAllFormat)
	}

	// 2. Setup Output Directory
	outDir := exportAllOutput
	if outDir == "" {
		outDir = filepath.Join(appVault.RootPath, "exports", exportAllFormat)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// 3. Create Filter (Shared)
	tmpFilter, err := os.CreateTemp("", "lx-filter-*.lua")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFilter.Name())
	tmpFilter.WriteString(assets.LinksFilter)
	tmpFilter.Close()

	// 4. Get All Notes
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	total := len(headers)
	if total == 0 {
		fmt.Println(ui.FormatWarning("No notes to export."))
		return nil
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Exporting %d notes to %s...", total, outDir)))
	fmt.Println(ui.RenderKeyValue("Format", exportAllFormat))
	fmt.Println(ui.RenderKeyValue("Workers", fmt.Sprintf("%d", exportAllJobs)))
	fmt.Println()

	// 5. Worker Pool Setup
	jobs := make(chan domain.NoteHeader, total)
	results := make(chan error, total)
	var wg sync.WaitGroup

	// Start Workers
	for i := 0; i < exportAllJobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for h := range jobs {
				// Reusing the convertNote function from cmd/export.go
				// Since they are in the same package (cmd), it is accessible.
				err := convertNote(h, outDir, tmpFilter.Name(), profile)
				results <- err
			}
		}()
	}

	// Queue Jobs
	for _, h := range headers {
		jobs <- h
	}
	close(jobs)

	// Wait for completion in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// 6. Progress Loop
	success := 0
	failed := 0

	// Simple progress bar
	for err := range results {
		if err != nil {
			failed++
			// Optional: print errors verbosely?
			// For now, keep it clean like build-all
		} else {
			success++
		}

		// Update progress line
		fmt.Printf("\r%s Progress: %d/%d (%d failed)",
			ui.StyleAccent.Render("➤"),
			success+failed,
			total,
			failed)
	}

	fmt.Println()
	fmt.Println()

	if failed > 0 {
		fmt.Println(ui.FormatWarning(fmt.Sprintf("Completed with %d errors.", failed)))
	} else {
		fmt.Println(ui.FormatSuccess("Export complete!"))
	}

	return nil
}
