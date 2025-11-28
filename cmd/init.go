package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"lx/pkg/ui"
	"lx/pkg/vault"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the lx vault",
	Long: `Initialize the lx vault directory structure.

This creates the managed vault at ~/.local/share/lx/ with the following structure:
  - notes/      : Your LaTeX source files
  - templates/  : Your .sty template files
  - cache/      : Build artifacts (PDFs, logs, etc.)
  - config.yaml : Global configuration`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Create vault instance
	v, err := vault.New()
	if err != nil {
		fmt.Println(ui.FormatError("Failed to determine vault location"))
		return err
	}

	// Check if already initialized
	if v.Exists() {
		fmt.Println(ui.FormatWarning("Vault already initialized"))
		fmt.Println(ui.FormatMuted("Location: " + v.RootPath))
		return nil
	}

	// Initialize the vault
	fmt.Println(ui.FormatRocket("Initializing lx vault..."))
	fmt.Println()

	if err := v.Initialize(); err != nil {
		fmt.Println(ui.FormatError("Failed to initialize vault"))
		return err
	}

	// Create default config
	if err := createDefaultConfig(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create default config: " + err.Error()))
		// Don't fail - config is optional
	}

	// Create .latexmkrc file
	if err := createLatexmkrc(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create .latexmkrc: " + err.Error()))
	} else {
		fmt.Println(ui.FormatSuccess("Editor config (.latexmkrc) created"))
	}

	// Success message
	fmt.Println(ui.FormatSuccess("Vault initialized successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Location", v.RootPath))
	fmt.Println()
	fmt.Println(ui.FormatInfo("Directory structure:"))
	fmt.Println(ui.FormatMuted("  notes/      - Your LaTeX notes (.tex files)"))
	fmt.Println(ui.FormatMuted("  templates/  - Your style files (.sty files)"))
	fmt.Println(ui.FormatMuted("  cache/      - Compiled PDFs and build artifacts"))
	fmt.Println()
	fmt.Println(ui.FormatInfo("Next steps:"))
	fmt.Println(ui.FormatMuted("  1. Create your first note: lx new \"My First Note\""))
	fmt.Println(ui.FormatMuted("  2. List all notes: lx list"))
	fmt.Println(ui.FormatMuted("  3. Build a note: lx build <query>"))

	return nil
}

func createDefaultConfig(v *vault.Vault) error {
	// Simple default config
	defaultConfig := `# LX Configuration
# This file is optional - all settings have sensible defaults

# Default template to use when creating new notes
# default_template: ""

# Default editor (uses $EDITOR environment variable if not set)
# editor: ""

# Number of concurrent jobs for build-all
# max_workers: 4
`

	configDir := filepath.Dir(v.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(v.ConfigPath, []byte(defaultConfig), 0644)
}

func createLatexmkrc(v *vault.Vault) error {
	// This file tells your editor (VS Code, VimTeX, etc.) to:
	// 1. Output all build files to ../cache
	// 2. Look in ../templates for .sty files
	content := `# LX Editor Configuration
# This ensures your editor uses the shared cache and templates.

$out_dir = '../cache';
$pdf_mode = 1;

# Add templates folder to TEXINPUTS (recursively)
my $templates_path = '../templates//';

if ($^O eq 'MSWin32') {
    $ENV{'TEXINPUTS'} = $templates_path . ';' . $ENV{'TEXINPUTS'};
} else {
    $ENV{'TEXINPUTS'} = $templates_path . ':' . $ENV{'TEXINPUTS'};
}
`
	// We place this INSIDE the notes folder because that is where the editor runs latexmk
	path := filepath.Join(v.NotesPath, ".latexmkrc")
	return os.WriteFile(path, []byte(content), 0644)
}
