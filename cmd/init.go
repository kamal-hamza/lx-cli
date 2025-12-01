package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
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

	// Create .latexmkrc file for notes directory
	if err := createLatexmkrc(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create .latexmkrc: " + err.Error()))
	} else {
		fmt.Println(ui.FormatSuccess("Editor config (.latexmkrc) created"))
	}

	// Create .latexmkrc file for templates directory
	if err := createTemplatesLatexmkrc(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create templates .latexmkrc: " + err.Error()))
	} else {
		fmt.Println(ui.FormatSuccess("Templates editor config (.latexmkrc) created"))
	}

	// Create .gitignore file
	if err := createGitignore(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create .gitignore: " + err.Error()))
	} else {
		fmt.Println(ui.FormatSuccess("Git ignore file (.gitignore) created"))
	}

	// Create Zed workspace settings
	if err := createZedSettings(v); err != nil {
		fmt.Println(ui.FormatWarning("Failed to create Zed settings: " + err.Error()))
	} else {
		fmt.Println(ui.FormatSuccess("Zed workspace settings (.zed/settings.json) created"))
	}

	// Check/Install Pandoc
	if err := checkAndInstallPandoc(); err != nil {
		// Warn but don't fail init, as it's optional for basic usage
		fmt.Println(ui.FormatWarning("Pandoc check skipped: " + err.Error()))
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
	fmt.Println(ui.FormatMuted("  assets/     - Static assets (images, bibliographies, etc.)"))
	fmt.Println(ui.FormatMuted("  .zed/       - Zed editor workspace settings"))
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
	// We need to add BOTH templates and assets to the path
	content := `# LX Editor Configuration
$out_dir = '../cache';
$pdf_mode = 1;

# Recursive search for templates and assets
my $templates_path = '../templates//';
my $assets_path = '../assets//';

# Platform-specific path separator
my $sep = ($^O eq 'MSWin32') ? ';' : ':';

if ($^O eq 'MSWin32') {
    $ENV{'TEXINPUTS'} = $templates_path . $sep . $assets_path . $sep . $ENV{'TEXINPUTS'};
} else {
    $ENV{'TEXINPUTS'} = $templates_path . $sep . $assets_path . $sep . $ENV{'TEXINPUTS'};
}
`
	path := filepath.Join(v.NotesPath, ".latexmkrc")
	return os.WriteFile(path, []byte(content), 0644)
}

func createTemplatesLatexmkrc(v *vault.Vault) error {
	// For templates directory, redirect output to cache and include assets
	content := `# LX Editor Configuration (Templates Directory)
$out_dir = '../cache';
$pdf_mode = 1;

# Recursive search for assets
my $assets_path = '../assets//';

# Platform-specific path separator
my $sep = ($^O eq 'MSWin32') ? ';' : ':';

if ($^O eq 'MSWin32') {
    $ENV{'TEXINPUTS'} = $assets_path . $sep . $ENV{'TEXINPUTS'};
} else {
    $ENV{'TEXINPUTS'} = $assets_path . $sep . $ENV{'TEXINPUTS'};
}
`
	path := filepath.Join(v.TemplatesPath, ".latexmkrc")
	return os.WriteFile(path, []byte(content), 0644)
}

func createGitignore(v *vault.Vault) error {
	// Ignore cache, OS files, and common editor configs
	content := `# LX Vault
cache/
dist/
build/

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Editor directories and files
.idea/
.vscode/
*.swp
*.swo
*~
*.bak

# Keep Zed workspace settings (for lx-ls + texlab integration)
!.zed/

# LaTeX generated files (in case they leak out of cache)
*.aux
*.log
*.out
*.toc
*.fls
*.fdb_latexmk
*.synctex.gz
`
	path := filepath.Join(v.RootPath, ".gitignore")
	return os.WriteFile(path, []byte(content), 0644)
}

func createZedSettings(v *vault.Vault) error {
	// Create Zed workspace settings to enable both lx-ls and texlab
	content := `{
  "lsp": {
    "texlab": {
      "initialization_options": {
        "diagnostics": {
          "ignoredPatterns": []
        }
      }
    }
  },
  "languages": {
    "LaTeX": {
      "language_servers": ["lx-ls", "texlab"]
    }
  }
}
`
	// Create .zed directory
	zedDir := filepath.Join(v.RootPath, ".zed")
	if err := os.MkdirAll(zedDir, 0755); err != nil {
		return fmt.Errorf("failed to create .zed directory: %w", err)
	}

	// Write settings.json
	settingsPath := filepath.Join(zedDir, "settings.json")
	return os.WriteFile(settingsPath, []byte(content), 0644)
}
