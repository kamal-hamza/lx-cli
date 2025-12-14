package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/config"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage command aliases",
	Long: `Manage user-defined command aliases.

Aliases allow you to create shortcuts for frequently-used commands or command sequences.

Examples:
  lx alias list                        # List all aliases
  lx alias add hw "new -t homework"    # Create 'hw' alias
  lx alias remove hw                    # Remove 'hw' alias`,
}

var aliasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all defined aliases",
	Long:  `Display all user-defined command aliases.`,
	RunE:  runAliasList,
}

var aliasAddCmd = &cobra.Command{
	Use:   "add <name> <command>",
	Short: "Add a new alias",
	Long: `Add a new command alias to your configuration.

The alias name should be a single word (no spaces).
The command can be any valid lx command with arguments.

Examples:
  lx alias add hw "new -t homework"
  lx alias add today "list -s modified -r"
  lx alias add backup "export -f json -o ~/backup.json"
  lx alias add qp "new -t quick-note -e"`,
	Args: cobra.ExactArgs(2),
	RunE: runAliasAdd,
}

var aliasRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete", "del"},
	Short:   "Remove an alias",
	Long:    `Remove a command alias from your configuration.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runAliasRemove,
}

func init() {
	aliasCmd.AddCommand(aliasListCmd)
	aliasCmd.AddCommand(aliasAddCmd)
	aliasCmd.AddCommand(aliasRemoveCmd)
}

func runAliasList(cmd *cobra.Command, args []string) error {
	cfg := appConfig

	if len(cfg.Aliases) == 0 {
		fmt.Println(ui.FormatInfo("No aliases defined"))
		fmt.Println(ui.FormatMuted("\nTo add an alias, use:"))
		fmt.Println(ui.FormatMuted("  lx alias add <name> <command>"))
		return nil
	}

	// Sort aliases by name for consistent output
	names := make([]string, 0, len(cfg.Aliases))
	for name := range cfg.Aliases {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println(ui.StyleTitle.Render("Command Aliases"))
	fmt.Println()

	maxNameLen := 0
	for _, name := range names {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	for _, name := range names {
		command := cfg.Aliases[name]
		padding := strings.Repeat(" ", maxNameLen-len(name))
		fmt.Printf("  %s%s  →  %s\n",
			ui.StyleSuccess.Render(name),
			padding,
			ui.FormatMuted(command))
	}

	fmt.Println()
	fmt.Printf(ui.FormatInfo("Total: %d alias(es)"), len(cfg.Aliases))
	fmt.Println()

	return nil
}

func runAliasAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1]

	// Validate alias name
	if err := validateAliasName(name); err != nil {
		return err
	}

	// Check if alias conflicts with existing commands
	if isReservedCommand(name) {
		return fmt.Errorf("cannot create alias '%s': conflicts with existing command", name)
	}

	cfg := appConfig

	// Check if alias already exists
	if existing, ok := cfg.Aliases[name]; ok {
		fmt.Println(ui.FormatWarning(fmt.Sprintf("Alias '%s' already exists: %s", name, existing)))
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println(ui.FormatInfo("Cancelled"))
			return nil
		}
	}

	// Add alias
	cfg.Aliases[name] = command

	// Save config
	if err := cfg.Save(appVault.ConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("✓ Created alias: %s → %s", name, command)))
	fmt.Println(ui.FormatInfo("Restart lx or reload your shell for the alias to take effect"))

	return nil
}

func runAliasRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg := appConfig

	// Check if alias exists
	command, ok := cfg.Aliases[name]
	if !ok {
		return fmt.Errorf("alias '%s' not found", name)
	}

	// Remove alias
	delete(cfg.Aliases, name)

	// Save config
	if err := cfg.Save(appVault.ConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("✓ Removed alias: %s → %s", name, command)))

	return nil
}

// validateAliasName checks if an alias name is valid
func validateAliasName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("alias name cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("alias name cannot start with '-'")
	}

	// Check for invalid characters
	for _, ch := range name {
		if !isValidAliasChar(ch) {
			return fmt.Errorf("alias name contains invalid character: %c", ch)
		}
	}

	return nil
}

// isValidAliasChar checks if a character is valid for an alias name
func isValidAliasChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' || ch == '_'
}

// isReservedCommand checks if a name conflicts with existing commands
func isReservedCommand(name string) bool {
	reserved := map[string]bool{
		"new":        true,
		"list":       true,
		"open":       true,
		"edit":       true,
		"delete":     true,
		"build":      true,
		"build-all":  true,
		"init":       true,
		"purge":      true,
		"version":    true,
		"git":        true,
		"clone":      true,
		"sync":       true,
		"rename":     true,
		"doctor":     true,
		"stats":      true,
		"clean":      true,
		"config":     true,
		"tag":        true,
		"graph":      true,
		"grep":       true,
		"daily":      true,
		"links":      true,
		"explore":    true,
		"export":     true,
		"attach":     true,
		"watch":      true,
		"todo":       true,
		"reindex":    true,
		"daemon":     true,
		"assets":     true,
		"export-all": true,
		"migrate":    true,
		"alias":      true,
		"dashboard":  true,
		"dash":       true,
		"help":       true,
	}

	return reserved[name]
}

// expandAlias expands an alias command string, supporting variable substitution
func expandAlias(aliasCmd string, args []string) []string {
	// Simple implementation: just append args to the alias command
	// Split the alias command into parts
	parts := strings.Fields(aliasCmd)

	// Support $1, $2, etc. variable substitution
	for i := range parts {
		// Replace $1, $2, etc. with actual arguments
		for j, arg := range args {
			placeholder := fmt.Sprintf("$%d", j+1)
			parts[i] = strings.ReplaceAll(parts[i], placeholder, arg)
		}
		// Replace $@ with all arguments
		parts[i] = strings.ReplaceAll(parts[i], "$@", strings.Join(args, " "))
	}

	// Append any remaining args that weren't substituted
	hasSubstitution := strings.Contains(aliasCmd, "$")

	if !hasSubstitution {
		parts = append(parts, args...)
	}

	return parts
}

// TryResolveAlias attempts to resolve a command as an alias
// Returns the expanded command parts and true if it was an alias, or nil and false if not
func TryResolveAlias(cfg *config.Config, cmdName string, args []string) ([]string, bool) {
	if cfg == nil || cfg.Aliases == nil {
		return nil, false
	}

	aliasCmd, ok := cfg.Aliases[cmdName]
	if !ok {
		return nil, false
	}

	expanded := expandAlias(aliasCmd, args)
	return expanded, true
}
