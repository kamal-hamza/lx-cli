package cmd

import (
	"fmt"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git [args...]",
	Short: "Run git commands within the lx notes directory",
	Long: `Run git commands within the lx notes directory.

Examples:
  # Check the status of the lx notes repository
  lx git status

  # Commit changes with a message
  lx git commit -am "Updated notes"

  # Push changes to the remote repository
  lx git push`,
	Args: cobra.ArbitraryArgs,
	RunE: runGit,
}

func runGit(cmd *cobra.Command, args []string) error {
	// Check if app vault exists
	if !appVault.Exists() {
		fmt.Println(ui.FormatError("Vault not Initialized"))
		return nil
	}

	c := exec.Command("git", args...)
	c.Dir = appVault.RootPath
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return err
	}

	return nil
}
