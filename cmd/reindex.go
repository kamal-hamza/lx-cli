package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild the vault index and connection graph",
	Long: `Rebuild the vault index by scanning all notes for metadata and connections.

This command:
  1. Scans all .tex files in the vault
  2. Extracts metadata (title, date, tags)
  3. Detects LaTeX links (\input, \ref, \cite, etc.)
  4. Calculates backlinks by inverting connections
  5. Saves the index to index.json

The index enables fast lookups and graph visualization without scanning files.`,
	RunE: runReindex,
}

func runReindex(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	fmt.Println(ui.FormatRocket("Reindexing vault..."))
	fmt.Println()
	// Record start time
	startTime := time.Now()
	// Execute reindex
	indexerService := services.NewIndexerService(noteRepo, appVault.RootPath)
	req := services.ReindexRequest{}
	resp, err := indexerService.Execute(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Reindex failed"))
		return err
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Success message
	fmt.Println(ui.FormatSuccess("Index rebuilt successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Total Notes", fmt.Sprintf("%d", resp.TotalNotes)))
	fmt.Println(ui.RenderKeyValue("Total Connections", fmt.Sprintf("%d", resp.TotalConnections)))
	fmt.Println(ui.RenderKeyValue("Duration", duration.Round(time.Millisecond).String()))
	fmt.Println()
	fmt.Println(ui.FormatMuted("Index saved to: " + appVault.RootPath + "/index.json"))

	return nil
}
