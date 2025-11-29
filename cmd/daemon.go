package cmd

import (
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
	daemonQuiet bool
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run background daemon for auto-reindexing",
	Long: `Run a background daemon that watches for file changes and automatically reindexes.

This command monitors the notes directory for:
  - New .tex files created
  - Existing .tex files modified
  - .tex files deleted

When changes are detected, it automatically rebuilds the index to keep
connections, backlinks, and metadata up-to-date.

Use --quiet to suppress reindex notifications.`,
	RunE: runDaemon,
}

func init() {
	daemonCmd.Flags().BoolVarP(&daemonQuiet, "quiet", "q", false, "Suppress reindex notifications")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	ctx := getContext()

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

	if !daemonQuiet {
		fmt.Println(ui.FormatRocket("Starting lx daemon..."))
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

		if !daemonQuiet {
			fmt.Println(ui.FormatInfo("File changes detected, reindexing..."))
		}

		indexerService := services.NewIndexerService(noteRepo, appVault.IndexPath())
		req := services.ReindexRequest{}
		resp, err := indexerService.Execute(ctx, req)
		if err != nil {
			if !daemonQuiet {
				fmt.Println(ui.FormatError("Reindex failed: " + err.Error()))
			}
			log.Printf("Reindex error: %v", err)
			return
		}

		if !daemonQuiet {
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
			if !daemonQuiet {
				fmt.Println()
				fmt.Println(ui.FormatMuted("Daemon stopped"))
			}
			return nil
		}
	}
}
