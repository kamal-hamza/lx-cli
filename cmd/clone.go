package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"lx/pkg/ui"
	"lx/pkg/vault"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone [url]",
	Short: "Clone a vault from a remote repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runClone,
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]

	// Create a vault instance to get paths (ignoring errors as dirs might not exist yet)
	v, _ := vault.New()

	// 1. Safety Check
	if v.Exists() {
		return fmt.Errorf("vault already exists at %s. Please delete it before cloning", v.RootPath)
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Cloning from %s...", repoURL)))
	fmt.Println(ui.RenderKeyValue("Destination", v.RootPath))

	// 2. Run Clone
	c := exec.Command("git", "clone", repoURL, v.RootPath)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess("Vault cloned successfully!"))
	fmt.Println(ui.FormatInfo("Run 'lx list' to view your notes."))

	return nil
}
