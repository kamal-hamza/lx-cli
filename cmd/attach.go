package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:     "attach [file]",
	Short:   "Import an asset with interactive metadata",
	Aliases: []string{"a"},
	Long: `Import an attachment into the vault.

Checks for duplicates automatically. If the file already exists (even under a
different name), it reuses the existing file.`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func runAttach(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	srcPath := args[0]

	svc := services.NewAttachmentService(appVault, assetRepo)

	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Attaching %s...", filepath.Base(absPath))))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// 1. Interactive Prompt: Name
	fmt.Print(ui.StyleInfo.Render("? Enter a name (slug): "))
	nameInput, _ := reader.ReadString('\n')
	nameInput = strings.TrimSpace(nameInput)
	if nameInput == "" {
		// Default to filename if empty (fallback logic)
		nameInput = strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		fmt.Printf("  Using default: %s\n", nameInput)
	}

	// 2. Interactive Prompt: Description (Mandatory)
	var descInput string
	for {
		fmt.Print(ui.StyleInfo.Render("? Enter description (required): "))
		descInput, _ = reader.ReadString('\n')
		descInput = strings.TrimSpace(descInput)
		if descInput != "" {
			break
		}
		fmt.Println(ui.FormatWarning("Description cannot be empty."))
	}

	fmt.Println()

	// 3. Store File (Handling 3 return values for duplicate detection)
	filename, isDuplicate, err := svc.Store(ctx, absPath, nameInput, descInput)
	if err != nil {
		return err
	}

	// 4. Output
	latexSnippet := fmt.Sprintf("\\includegraphics[width=0.8\\linewidth]{%s}", filename)

	if isDuplicate {
		// Duplicate Found Message
		fmt.Println(ui.FormatWarning("Duplicate image found in vault!"))
		fmt.Println(ui.FormatInfo("Reusing existing file: " + ui.StyleBold.Render(filename)))
		fmt.Println(ui.FormatMuted("(No new file was created)"))
	} else {
		// Success Message
		fmt.Println(ui.FormatSuccess("Asset stored: " + filename))
		fmt.Println(ui.FormatMuted("Description saved."))
	}

	fmt.Println()
	fmt.Println(ui.FormatInfo("LaTeX Code (Copied):"))
	fmt.Println(ui.StyleBold.Render(latexSnippet))

	if err := clipboard.WriteAll(latexSnippet); err != nil {
		fmt.Println(ui.FormatMuted("(Clipboard access failed)"))
	}

	return nil
}
