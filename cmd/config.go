package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit the lx configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := appVault.ConfigPath

		// Ensure it exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("config file not found at %s", path)
		}

		fmt.Println(ui.FormatInfo("Opening config: " + path))

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}
