package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"lx/pkg/ui"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your lx installation",
	Long: `Diagnose issues with your LX setup.

Checks for:
  - Vault directory integrity
  - Configuration file existence
  - Required system dependencies (latexmk, git)
  - Environment variables`,
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

	checkStep("Templates Directory", func() error {
		if _, err := os.Stat(appVault.TemplatesPath); os.IsNotExist(err) {
			return fmt.Errorf("missing at %s", appVault.TemplatesPath)
		}
		return nil
	})

	checkStep("Cache Directory", func() error {
		if _, err := os.Stat(appVault.CachePath); os.IsNotExist(err) {
			return fmt.Errorf("missing at %s", appVault.CachePath)
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
		path, err := exec.LookPath("latexmk")
		if err != nil {
			return fmt.Errorf("not found in PATH")
		}
		// Optional: Check version?
		// For now, just knowing it exists is enough.
		_ = path
		return nil
	})

	checkStep("git (Version Control)", func() error {
		_, err := exec.LookPath("git")
		if err != nil {
			return fmt.Errorf("not found (required for sync/clone)")
		}
		return nil
	})

	// 4. Check Environment
	checkStep("EDITOR Variable", func() error {
		if os.Getenv("EDITOR") == "" {
			return fmt.Errorf("not set (will fall back to 'vi')")
		}
		return nil
	})

	fmt.Println()
	fmt.Println(ui.FormatInfo("Diagnosis complete."))

	// Check .latexmkrc file
	checkStep("Editor Config (.latexmkrc)", func() error {
		path := filepath.Join(appVault.NotesPath, ".latexmkrc")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("missing (your editor might clutter the notes folder)")
		}
		return nil
	})

	// Check pandoc
	checkStep("pandoc (Export Tool)", func() error {
		return checkAndInstallPandoc()
	})

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
		// Print error with indentation for better readability
		fmt.Printf("%s %s\n", ui.FormatError("✘"), name)
		fmt.Printf("    %s\n", ui.StyleMuted.Render("Error: "+err.Error()))
	}
}
