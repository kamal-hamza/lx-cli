package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var watchCmd = &cobra.Command{
	Use:     "watch [query]",
	Aliases: []string{"w"},
	Short:   "Live preview a note (auto-build on save) (alias: w)",
	Long: `Continuously rebuild a note whenever you save it.

This uses a file watcher to trigger the Preprocess -> Compile cycle
automatically.

It attempts to open the PDF viewer on the first successful build.
Subsequent builds will update the file in place (silent reload).

REQUIREMENT: Use a PDF viewer that supports 'auto-reload on change':
  - macOS: Skim (enable "Check for file changes" in settings)
  - Windows: SumatraPDF
  - Linux: Zathura`,
	RunE: runWatch,
}

func runWatch(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Find the Note (Standard selection logic)
	var selectedNote *domain.NoteHeader

	if len(args) == 0 {
		req := services.ListRequest{SortBy: "date"}
		resp, err := listService.Execute(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found."))
			return nil
		}

		idx, err := fuzzyfinder.Find(
			resp.Notes,
			func(i int) string { return resp.Notes[i].Title },
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				return fmt.Sprintf("Watch Mode\n\nTitle: %s\nSlug: %s",
					resp.Notes[i].Title, resp.Notes[i].Slug)
			}),
		)
		if err != nil {
			return nil
		}
		selectedNote = &resp.Notes[idx]
	} else {
		req := services.SearchRequest{Query: args[0]}
		resp, err := listService.Search(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found matching: " + args[0]))
			return nil
		}
		selectedNote = &resp.Notes[0]
	}

	// 2. Start Watcher
	fmt.Println(ui.FormatRocket("Starting watcher for: " + ui.StyleBold.Render(selectedNote.Title)))
	fmt.Println(ui.FormatMuted("Press Ctrl+C to stop"))
	fmt.Println()

	// State to track if we've opened the viewer
	hasOpenedViewer := false
	pdfPath := appVault.GetCachePath(selectedNote.Slug + ".pdf")

	// Helper to handle build and opening
	handleBuild := func() {
		result, err := triggerBuildWithDetails(selectedNote.Slug)

		if err != nil {
			// Show a clean error message
			fmt.Println(ui.FormatError("Build failed: " + err.Error()))

			// If we have parsed errors, show them
			if result != nil && result.Parsed != nil && len(result.Parsed.Errors) > 0 {
				fmt.Println(ui.FormatMuted("\nErrors found:"))
				fmt.Println(result.Parsed.FormatIssues())
			}
		} else {
			// Success case
			timestamp := time.Now().Format("15:04:05")

			if result != nil && result.Parsed != nil {
				summary := result.Parsed.GetSummary()
				fmt.Printf("\r%s %s at %s\n", summary, selectedNote.Slug, timestamp)

				// Show warnings if any
				if len(result.Parsed.Warnings) > 0 {
					fmt.Println(ui.FormatMuted(fmt.Sprintf("  ⚠️  %d warning(s)", len(result.Parsed.Warnings))))
				}
			} else {
				fmt.Printf("\r%s Rebuilt %s at %s\n",
					ui.FormatSuccess("✓"),
					selectedNote.Slug,
					timestamp)
			}

			// Open viewer on first success
			if !hasOpenedViewer {
				fmt.Println(ui.FormatInfo("Opening PDF viewer..."))
				if err := OpenFileWithDefaultApp(pdfPath); err != nil {
					fmt.Println(ui.FormatWarning("Failed to open PDF: " + err.Error()))
				}
				hasOpenedViewer = true
			}
		}
	}

	// Initial Build
	handleBuild()

	// 3. Setup Watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the specific note file
	notePath := appVault.GetNotePath(selectedNote.Filename)
	if err := watcher.Add(notePath); err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	// Debounce setup
	var debounceTimer *time.Timer
	const debounceDuration = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Handle Write events (and Rename/Chmod which some editors do on save)
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Chmod) {
				// Cancel previous timer if active
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Schedule new build
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					// We must re-add the watcher if the editor used "atomic save" (Rename/Move)
					// Verify file still exists and re-watch if needed
					// Ignoring error here as file might temporarily disappear during save
					watcher.Add(notePath)

					handleBuild()
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)

		case <-ctx.Done():
			return nil
		}
	}
}

// triggerBuild wraps the service call to keep the main loop clean
func triggerBuild(slug string) error {
	result, err := triggerBuildWithDetails(slug)
	if err != nil {
		return err
	}
	if result != nil && !result.Success {
		return fmt.Errorf("build failed")
	}
	return nil
}

// triggerBuildWithDetails returns detailed build results for better error reporting
func triggerBuildWithDetails(slug string) (*services.BuildResultDetails, error) {
	ctx := getContext()
	req := services.BuildRequest{Slug: slug}

	// This now calls the BuildService, which triggers Preprocessor -> Compiler
	result, err := buildService.ExecuteWithDetails(ctx, req)
	if err != nil {
		return result, err
	}

	return result, nil
}
