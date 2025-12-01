package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	purgeForce bool
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete the entire vault and all its contents",
	Long: `Delete the entire vault directory and all its contents.

This is a destructive operation that will permanently delete:
  - All notes in the vault
  - All templates
  - All assets (images, PDFs, etc.)
  - All cached PDFs
  - The index and all metadata
  - Configuration files

This action cannot be undone. Use with extreme caution.

Examples:
  # Purge the vault with confirmation prompt
  lx purge

  # Force purge without confirmation (dangerous!)
  lx purge --force`,
	RunE: runPurge,
}

func init() {
	purgeCmd.Flags().BoolVarP(&purgeForce, "force", "f", false, "Skip confirmation prompt (dangerous)")
}

func runPurge(cmd *cobra.Command, args []string) error {
	// Check if vault exists
	if !appVault.Exists() {
		fmt.Println(ui.FormatWarning("Vault does not exist."))
		fmt.Println(ui.FormatInfo("Vault location: " + appVault.RootPath))
		return nil
	}

	// Display what will be deleted
	fmt.Println(ui.StyleError.Render("⚠️  WARNING: DESTRUCTIVE OPERATION ⚠️"))
	fmt.Println()
	fmt.Println(ui.FormatWarning("You are about to permanently delete the entire vault:"))
	fmt.Printf("  %s %s\n", ui.StyleBold.Render("Location:"), appVault.RootPath)
	fmt.Println()
	fmt.Println("This will delete:")
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("All notes"))
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("All templates"))
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("All assets"))
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("All cached files"))
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("The index"))
	fmt.Printf("  • %s\n", ui.StyleMuted.Render("Configuration"))
	fmt.Println()
	fmt.Println(ui.FormatError("⚠️  THIS ACTION CANNOT BE UNDONE ⚠️"))
	fmt.Println()

	// Check if force flag is set
	if !purgeForce {
		reader := bufio.NewReader(os.Stdin)

		// First confirmation
		var firstConfirmed bool
		for {
			fmt.Print(ui.StyleError.Render("Are you absolutely sure you want to delete the vault? (yes/no): "))

			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input. Please type 'yes' or 'no'."))
				continue
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response == "yes" {
				firstConfirmed = true
				break
			} else if response == "no" {
				firstConfirmed = false
				break
			} else {
				fmt.Println(ui.FormatWarning("Please type 'yes' or 'no' (full words required)."))
			}
		}

		if !firstConfirmed {
			fmt.Println(ui.FormatInfo("Purge cancelled."))
			return nil
		}

		fmt.Println()

		// Second confirmation - require typing vault path
		var secondConfirmed bool
		for {
			fmt.Printf("%s %s\n",
				ui.StyleError.Render("To confirm, type the vault path:"),
				ui.StyleBold.Render(appVault.RootPath))
			fmt.Print(ui.StyleError.Render("> "))

			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println(ui.FormatWarning("Invalid input."))
				continue
			}

			response = strings.TrimSpace(response)
			if response == appVault.RootPath {
				secondConfirmed = true
				break
			} else if response == "" {
				secondConfirmed = false
				break
			} else {
				fmt.Println(ui.FormatWarning("Path does not match. Please try again or press Enter to cancel."))
			}
		}

		if !secondConfirmed {
			fmt.Println(ui.FormatInfo("Purge cancelled."))
			return nil
		}
	}

	fmt.Println()
	fmt.Println(ui.FormatInfo("Purging vault..."))

	// Delete the vault directory
	if err := os.RemoveAll(appVault.RootPath); err != nil {
		fmt.Println(ui.FormatError("Failed to delete vault: " + err.Error()))
		return err
	}

	// Also try to remove config file if it exists
	configDir := ""
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = xdgConfig + "/lx"
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		if appData := os.Getenv("APPDATA"); appData != "" {
			configDir = appData + "/lx-config"
		} else {
			configDir = homeDir + "/.config/lx"
		}
	}

	if configDir != "" {
		if _, err := os.Stat(configDir); err == nil {
			if err := os.RemoveAll(configDir); err != nil {
				fmt.Println(ui.FormatWarning("Warning: Failed to delete config directory: " + err.Error()))
			}
		}
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess("✓ Vault purged successfully"))
	fmt.Println(ui.FormatInfo("The vault has been permanently deleted."))
	fmt.Println(ui.FormatInfo("To create a new vault, run: lx init"))

	return nil
}
