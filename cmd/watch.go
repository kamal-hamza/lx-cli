package cmd

import (
	"fmt"
	"os"
	"os/exec"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var watchCmd = &cobra.Command{
	Use:   "watch [query]",
	Short: "Live preview a note (silent mode)",
	Long: `Continuously rebuild a note whenever you save it.

This runs 'latexmk' in background mode (-view=none).
It will NOT open or focus your PDF viewer.

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
	fmt.Println(ui.FormatRocket("Starting silent watcher for: " + ui.StyleBold.Render(selectedNote.Title)))
	fmt.Println(ui.FormatMuted("Open the PDF in Skim/SumatraPDF to see updates."))
	fmt.Println(ui.FormatMuted("Press Ctrl+C to stop"))
	fmt.Println()

	// Prepare latexmk command
	cmdArgs := []string{
		"-pvc",                     // Preview Continuously
		"-view=none",               // <--- THE FIX: Don't touch the viewer!
		"-pdf",                     // Output PDF
		"-interaction=nonstopmode", // Don't halt on errors
		"-file-line-error",         // Better error messages
		"-output-directory=" + appVault.CachePath,
		appVault.GetNotePath(selectedNote.Filename),
	}

	c := exec.Command("latexmk", cmdArgs...)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	// Inject Environment
	env := os.Environ()
	env = append(env, "TEXINPUTS="+appVault.GetTexInputsEnv())
	c.Env = env

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == -1 {
			return nil
		}
		return fmt.Errorf("watcher stopped: %w", err)
	}

	return nil
}
