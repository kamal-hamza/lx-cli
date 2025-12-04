package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	reindexWatch bool
	reindexQuiet bool
)

var reindexCmd = &cobra.Command{
	Use:     "reindex",
	Aliases: []string{"ri"},
	Short:   "Rebuild the vault index and connection graph (alias: ri)",
	Long: `Rebuild the vault index by scanning all notes for metadata and connections.

This command:
  1. Scans all .tex files in the vault
  2. Extracts metadata (title, date, tags)
  3. Detects LaTeX links (\input, \ref, \cite, etc.)
  4. Calculates backlinks by inverting connections
  5. Saves the index to index.json

The index enables fast lookups and graph visualization without scanning files.

Use --watch to continuously monitor for file changes and auto-reindex.`,
	RunE: runReindex,
}

func init() {
	reindexCmd.Flags().BoolVarP(&reindexWatch, "watch", "w", false, "Watch for changes and auto-reindex")
	reindexCmd.Flags().BoolVarP(&reindexQuiet, "quiet", "q", false, "Suppress reindex notifications (only with --watch)")
}

func runReindex(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// If --watch flag is set, run in watch mode
	if reindexWatch {
		return runReindexWatch(ctx)
	}

	// Single reindex
	fmt.Println(ui.FormatRocket("Reindexing vault..."))
	fmt.Println()

	startTime := time.Now()
	indexerService := services.NewIndexerService(noteRepo, appVault.IndexPath())
	req := services.ReindexRequest{}
	resp, err := indexerService.Execute(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Reindex failed"))
		return err
	}

	duration := time.Since(startTime)

	fmt.Println(ui.FormatSuccess("Index rebuilt successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Total Notes", fmt.Sprintf("%d", resp.TotalNotes)))
	fmt.Println(ui.RenderKeyValue("Total Connections", fmt.Sprintf("%d", resp.TotalConnections)))
	fmt.Println(ui.RenderKeyValue("Duration", duration.Round(time.Millisecond).String()))
	fmt.Println()
	fmt.Println(ui.FormatMuted("Index saved to: " + appVault.IndexPath()))

	return nil
}

func runReindexWatch(ctx context.Context) error {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the notes directory
	if err := watcher.Add(appVault.NotesPath); err != nil {
		return fmt.Errorf("failed to watch notes directory: %w", err)
	}

	if !reindexQuiet {
		fmt.Println(ui.FormatRocket("Starting auto-reindex watcher..."))
		fmt.Println(ui.FormatMuted("Watching: " + appVault.NotesPath))
		fmt.Println(ui.FormatMuted("Press Ctrl+C to stop"))
		fmt.Println()
	}

	// Debounce timer to avoid excessive reindexing
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond
	needsReindex := false

	// Function to perform reindex
	doReindex := func() {
		if !needsReindex {
			return
		}
		needsReindex = false

		if !reindexQuiet {
			fmt.Println(ui.FormatInfo("File changes detected, reindexing..."))
		}

		indexerService := services.NewIndexerService(noteRepo, appVault.IndexPath())
		req := services.ReindexRequest{}
		resp, err := indexerService.Execute(ctx, req)
		if err != nil {
			if !reindexQuiet {
				fmt.Println(ui.FormatError("Reindex failed: " + err.Error()))
			}
			log.Printf("Reindex error: %v", err)
			return
		}

		if !reindexQuiet {
			fmt.Println(ui.FormatSuccess(fmt.Sprintf("Index updated (%d notes, %d connections)",
				resp.TotalNotes, resp.TotalConnections)))
		}
	}

	// Event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only care about .tex files
			if !strings.HasSuffix(event.Name, ".tex") {
				continue
			}

			// Filter out temporary/cache files
			baseName := filepath.Base(event.Name)
			if strings.HasPrefix(baseName, ".") || strings.HasPrefix(baseName, "~") {
				continue
			}

			// Check if it's a create, write, remove, or rename event
			if event.Has(fsnotify.Create) ||
				event.Has(fsnotify.Write) ||
				event.Has(fsnotify.Remove) ||
				event.Has(fsnotify.Rename) {

				needsReindex = true

				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, doReindex)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)

		case <-ctx.Done():
			if !reindexQuiet {
				fmt.Println()
				fmt.Println(ui.FormatMuted("Watcher stopped"))
			}
			return nil
		}
	}
}
