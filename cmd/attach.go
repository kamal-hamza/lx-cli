package cmd

import (
	"fmt"
	"path/filepath"

	"lx/internal/core/services"
	"lx/pkg/ui"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach [file]",
	Short: "Import an image or file into the vault",
	Long: `Import an attachment into the vault's assets directory.

Features:
- Deduplication: Files are renamed based on content hash.
- Auto-Clipboard: Copies the LaTeX code to your clipboard.
- Access: Assets are globally available to all notes.

Examples:
  lx attach ~/Downloads/graph.png
  lx attach screenshot.jpg`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func runAttach(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	srcPath := args[0]

	// 1. Initialize Service
	svc := services.NewAttachmentService(appVault)

	// 2. Validate Source
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Attaching %s...", filepath.Base(absPath))))

	// 3. Store File
	filename, err := svc.Store(ctx, absPath)
	if err != nil {
		return err
	}

	// 4. Generate Snippet
	// Standard LaTeX include. We don't need path because of TEXINPUTS
	latexSnippet := fmt.Sprintf("\\includegraphics[width=0.8\\linewidth]{%s}", filename)

	// 5. Success & Clipboard
	fmt.Println(ui.FormatSuccess("Asset stored: " + filename))
	fmt.Println()
	fmt.Println(ui.FormatInfo("LaTeX Code (Copied to Clipboard):"))
	fmt.Println(ui.StyleBold.Render(latexSnippet))

	// Try to write to clipboard (non-blocking if fails)
	if err := clipboard.WriteAll(latexSnippet); err != nil {
		fmt.Println(ui.FormatMuted("(Clipboard access failed, please copy manually)"))
	}

	return nil
}
