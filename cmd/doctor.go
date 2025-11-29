package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your lx installation",
	Long: `Diagnose issues with your LX setup.

Checks for:
  - Vault directory integrity (including assets)
  - Configuration file existence
  - Required tools (latexmk, pandoc, git)
  - Broken links`,
	Run: runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) {
	fmt.Println(ui.FormatTitle("🏥 LX Doctor"))
	fmt.Println()

	// 1. Check Vault Structure
	checkStep("Vault Directory", func() error {
		if !appVault.Exists() {
			return fmt.Errorf("not found at %s", appVault.RootPath)
		}
		return nil
	})

	checkStep("Notes Directory", func() error {
		if _, err := os.Stat(appVault.NotesPath); os.IsNotExist(err) {
			return fmt.Errorf("missing at %s", appVault.NotesPath)
		}
		return nil
	})

	checkStep("Assets Directory", func() error {
		if _, err := os.Stat(appVault.AssetsPath); os.IsNotExist(err) {
			return fmt.Errorf("missing at %s", appVault.AssetsPath)
		}
		return nil
	})

	checkStep("Asset Manifest", func() error {
		manifestPath := filepath.Join(appVault.AssetsPath, ".manifest.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			// This is not a fatal error, just a warning
			return fmt.Errorf("missing (will be created on next attach)")
		}
		return nil
	})

	// 2. Check Config
	checkStep("Configuration File", func() error {
		if _, err := os.Stat(appVault.ConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("missing at %s", appVault.ConfigPath)
		}
		return nil
	})

	// 3. Check Dependencies
	checkStep("latexmk (Compiler)", func() error {
		if _, err := exec.LookPath("latexmk"); err != nil {
			return fmt.Errorf("not found in PATH")
		}
		return nil
	})

	checkStep("pandoc (Export)", func() error {
		if _, err := exec.LookPath("pandoc"); err != nil {
			return fmt.Errorf("not found (required for 'lx export')")
		}
		return nil
	})

	checkStep("git (Sync)", func() error {
		if _, err := exec.LookPath("git"); err != nil {
			return fmt.Errorf("not found (required for 'lx sync')")
		}
		return nil
	})

	// 4. Check Environment
	checkStep("EDITOR Variable", func() error {
		if os.Getenv("EDITOR") == "" {
			return fmt.Errorf("not set (using fallback 'vi')")
		}
		return nil
	})

	fmt.Println()
	fmt.Println(ui.FormatInfo("Checking content integrity..."))

	// Check for broken links
	checkStep("Link Integrity", func() error {
		headers, _ := noteRepo.ListHeaders(getContext())
		slugMap := make(map[string]bool)
		for _, h := range headers {
			slugMap[h.Slug] = true
		}

		brokenCount := 0
		linkRegex := regexp.MustCompile(`\\ref\{([^}]+)\}`)

		for _, h := range headers {
			content, _ := os.ReadFile(appVault.GetNotePath(h.Filename))
			matches := linkRegex.FindAllStringSubmatch(string(content), -1)
			for _, m := range matches {
				targetSlug := m[1]
				if !slugMap[targetSlug] {
					if brokenCount == 0 {
						fmt.Println()
					}
					fmt.Printf("    %s -> %s (Missing)\n", h.Slug, targetSlug)
					brokenCount++
				}
			}
		}

		if brokenCount > 0 {
			return fmt.Errorf("found %d broken links", brokenCount)
		}
		return nil
	})
}

// checkStep runs a check function and prints the result nicely
func checkStep(name string, check func() error) {
	err := check()
	if err == nil {
		fmt.Printf("%s %s\n", ui.FormatSuccess("✔"), name)
	} else {
		// Differentiate between fatal errors and warnings?
		// For now, consistent error formatting
		fmt.Printf("%s %s\n", ui.FormatError("✘"), name)
		fmt.Printf("    %s\n", ui.StyleMuted.Render(err.Error()))
	}
}
