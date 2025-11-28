package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"lx/pkg/ui"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync notes with remote (Stash -> Pull -> Pop -> Push)",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.FormatInfo("Syncing vault..."))
	fmt.Println()

	// Helpers
	runQuiet := func(args ...string) error {
		c := exec.Command("git", args...)
		c.Dir = appVault.RootPath
		output, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		return nil
	}

	runInteractive := func(args ...string) error {
		c := exec.Command("git", args...)
		c.Dir = appVault.RootPath
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	// 1. Validation
	if _, err := os.Stat(appVault.RootPath + "/.git"); os.IsNotExist(err) {
		return fmt.Errorf("git not initialized. Run 'lx git init' first")
	}

	// 2. Check Dirty State
	fmt.Print(ui.StyleInfo.Render("Checking status... "))
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = appVault.RootPath
	output, _ := statusCmd.Output()
	isDirty := len(strings.TrimSpace(string(output))) > 0
	fmt.Println(ui.FormatSuccess("Done"))

	// 3. Stash
	if isDirty {
		fmt.Print(ui.StyleWarning.Render("Stashing local changes... "))
		if err := runQuiet("stash", "push", "-m", "lx-auto-stash"); err != nil {
			fmt.Println(ui.FormatError("Failed"))
			return err
		}
		fmt.Println(ui.FormatSuccess("Saved"))
	}

	// 4. Pull (Rebase)
	fmt.Println(ui.StyleInfo.Render("Pulling remote changes..."))
	if err := runInteractive("pull", "--rebase"); err != nil {
		if isDirty {
			fmt.Println(ui.FormatWarning("Pull failed. Restoring your changes..."))
			runQuiet("stash", "pop")
		}
		return fmt.Errorf("pull failed: %w", err)
	}

	// 5. Pop Stash
	if isDirty {
		fmt.Print(ui.StyleWarning.Render("Restoring local changes... "))
		if err := runQuiet("stash", "pop"); err != nil {
			fmt.Println(ui.FormatError("Conflict"))
			fmt.Println(ui.FormatWarning("Merge conflict detected."))
			fmt.Println(ui.FormatInfo("Please resolve conflicts manually in: " + appVault.RootPath))
			return fmt.Errorf("manual intervention required")
		}
		fmt.Println(ui.FormatSuccess("Restored"))
	}

	// 6. Commit & Push
	if isDirty {
		fmt.Print(ui.StyleInfo.Render("Committing... "))
		runQuiet("add", "-A")
		timestamp := time.Now().Format("2006-01-02 15:04")
		if err := runQuiet("commit", "-m", "lx sync: "+timestamp); err != nil {
			return err
		}
		fmt.Println(ui.FormatSuccess("Done"))
	}

	fmt.Println(ui.StyleInfo.Render("Pushing..."))
	if err := runInteractive("push"); err != nil {
		return fmt.Errorf("push failed")
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess("Vault synced successfully!"))
	return nil
}
