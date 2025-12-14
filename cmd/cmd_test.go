package cmd

import (
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/config"
)

// TestCommandStructure verifies that all commands are properly registered
func TestCommandStructure(t *testing.T) {
	commands := []string{
		"new", "list", "open", "edit", "delete", "build", "build-all",
		"init", "version", "git", "clone", "sync", "rename", "doctor",
		"stats", "clean", "config", "tag", "graph", "grep", "daily",
		"links", "explore", "export", "attach", "watch", "todo", "reindex",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{cmdName})
			if err != nil {
				t.Errorf("Command '%s' not found: %v", cmdName, err)
			}
			if cmd == nil {
				t.Errorf("Command '%s' is nil", cmdName)
			}
			if cmd.Use == "" {
				t.Errorf("Command '%s' has no Use field", cmdName)
			}
		})
	}
}

// TestRootCommandExists verifies the root command is properly configured
func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("Root command is nil")
	}

	if rootCmd.Use != "lx" {
		t.Errorf("Expected root command Use to be 'lx', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Root command Short description is empty")
	}
}

// TestCommandsHaveHelp verifies all commands have help text
func TestCommandsHaveHelp(t *testing.T) {
	commands := rootCmd.Commands()

	if len(commands) == 0 {
		t.Fatal("No commands registered")
	}

	for _, cmd := range commands {
		t.Run(cmd.Name(), func(t *testing.T) {
			if cmd.Short == "" {
				t.Errorf("Command '%s' has no Short description", cmd.Name())
			}
		})
	}
}

// TestServiceInitialization verifies services can be initialized with mocks
func TestServiceInitialization(t *testing.T) {
	mockNoteRepo := mocks.NewMockRepository()
	mockTemplateRepo := mocks.NewMockTemplateRepository()

	// Test CreateNoteService
	mockGitService := services.NewGitService("/tmp/test")
	mockConfig := &config.Config{}
	createNoteService := services.NewCreateNoteService(mockNoteRepo, mockTemplateRepo, mockGitService, mockConfig)
	if createNoteService == nil {
		t.Error("CreateNoteService is nil")
	}

	// Test CreateTemplateService
	createTemplateService := services.NewCreateTemplateService(mockTemplateRepo)
	if createTemplateService == nil {
		t.Error("CreateTemplateService is nil")
	}

	// Test ListService
	listService := services.NewListService(mockNoteRepo)
	if listService == nil {
		t.Error("ListService is nil")
	}
}

// TestSubcommands verifies specific subcommands exist
func TestSubcommands(t *testing.T) {
	tests := []struct {
		parent     string
		subcommand string
	}{
		{"tag", "add"},
		{"tag", "remove"},
	}

	for _, tt := range tests {
		t.Run(tt.parent+"_"+tt.subcommand, func(t *testing.T) {
			parentCmd, _, err := rootCmd.Find([]string{tt.parent})
			if err != nil {
				t.Fatalf("Parent command '%s' not found: %v", tt.parent, err)
			}

			found := false
			for _, cmd := range parentCmd.Commands() {
				if cmd.Name() == tt.subcommand {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Subcommand '%s' not found under '%s'", tt.subcommand, tt.parent)
			}
		})
	}
}

// TestFlagsExist verifies important flags are registered
func TestFlagsExist(t *testing.T) {
	tests := []struct {
		command  string
		flagName string
	}{
		{"list", "tag"},
		{"list", "sort"},
		{"list", "reverse"},
		{"list", "template"},
		{"new", "template"},
		{"new", "tags"},
		{"delete", "template"},
		{"build", "open"},
		{"build", "template"},
		{"rename", "template"},
	}

	for _, tt := range tests {
		t.Run(tt.command+"_"+tt.flagName, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.command})
			if err != nil {
				t.Fatalf("Command '%s' not found: %v", tt.command, err)
			}

			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag '--%s' not found on command '%s'", tt.flagName, tt.command)
			}
		})
	}
}

// TestCommandAliases verifies command aliases work
func TestCommandAliases(t *testing.T) {
	tests := []struct {
		alias      string
		command    string
		shouldFind bool
	}{
		// Phase 1 aliases
		{"n", "new", true},
		{"create", "new", true},
		{"ls", "list", true},
		{"o", "open", true},
		{"e", "edit", true},
		{"b", "build", true},
		{"d", "delete", true},
		{"rm", "delete", true},
		{"g", "grep", true},
		{"gg", "graph", true},
		{"s", "sync", true},
		{"mv", "rename", true},
		{"t", "tag", true},
		{"w", "watch", true},
		{"cl", "clean", true},
		{"dd", "daily", true},
		{"td", "todo", true},
		{"st", "stats", true},
		{"ri", "reindex", true},
		{"x", "explore", true},
		{"a", "attach", true},
		{"ex", "export", true},
		{"lk", "links", true},
	}

	for _, tt := range tests {
		t.Run(tt.alias+"->"+tt.command, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.alias})
			if tt.shouldFind {
				if err != nil {
					t.Errorf("Alias '%s' not found: %v", tt.alias, err)
				}
				if cmd == nil {
					t.Errorf("Command for alias '%s' is nil", tt.alias)
				}
				if cmd != nil && cmd.Name() != tt.command {
					t.Errorf("Alias '%s' resolved to '%s', expected '%s'", tt.alias, cmd.Name(), tt.command)
				}
			}
		})
	}
}

// TestAliasesInHelpText verifies that aliases are shown in help text
func TestAliasesInHelpText(t *testing.T) {
	tests := []struct {
		command string
		aliases []string
	}{
		{"new", []string{"n", "create"}},
		{"list", []string{"ls"}},
		{"open", []string{"o"}},
		{"edit", []string{"e"}},
		{"build", []string{"b"}},
		{"delete", []string{"d", "rm"}},
		{"grep", []string{"g"}},
		{"graph", []string{"gg"}},
		{"sync", []string{"s"}},
		{"rename", []string{"mv"}},
		{"tag", []string{"t"}},
		{"watch", []string{"w"}},
		{"clean", []string{"cl"}},
		{"daily", []string{"dd"}},
		{"todo", []string{"td"}},
		{"stats", []string{"st"}},
		{"reindex", []string{"ri"}},
		{"explore", []string{"x"}},
		{"attach", []string{"a"}},
		{"export", []string{"ex"}},
		{"links", []string{"lk"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.command})
			if err != nil {
				t.Fatalf("Command '%s' not found: %v", tt.command, err)
			}

			if cmd == nil {
				t.Fatalf("Command '%s' is nil", tt.command)
			}

			// Check that the command has the expected aliases
			if len(cmd.Aliases) != len(tt.aliases) {
				t.Errorf("Command '%s' has %d aliases, expected %d", tt.command, len(cmd.Aliases), len(tt.aliases))
			}

			// Check each alias exists
			for _, expectedAlias := range tt.aliases {
				found := false
				for _, actualAlias := range cmd.Aliases {
					if actualAlias == expectedAlias {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Command '%s' missing expected alias '%s'", tt.command, expectedAlias)
				}
			}
		})
	}
}

// TestAllAliasesResolveToCorrectCommand ensures aliases work bidirectionally
func TestAllAliasesResolveToCorrectCommand(t *testing.T) {
	aliasMap := map[string]string{
		"n":      "new",
		"create": "new",
		"ls":     "list",
		"o":      "open",
		"e":      "edit",
		"b":      "build",
		"d":      "delete",
		"rm":     "delete",
		"g":      "grep",
		"gg":     "graph",
		"s":      "sync",
		"mv":     "rename",
		"t":      "tag",
		"w":      "watch",
		"cl":     "clean",
		"dd":     "daily",
		"td":     "todo",
		"st":     "stats",
		"ri":     "reindex",
		"x":      "explore",
		"a":      "attach",
		"ex":     "export",
		"lk":     "links",
	}

	for alias, expectedCommand := range aliasMap {
		t.Run(alias, func(t *testing.T) {
			// Test that alias resolves to command
			cmdViaAlias, _, err := rootCmd.Find([]string{alias})
			if err != nil {
				t.Fatalf("Failed to find command via alias '%s': %v", alias, err)
			}

			// Test that full command name also works
			cmdViaName, _, err := rootCmd.Find([]string{expectedCommand})
			if err != nil {
				t.Fatalf("Failed to find command via name '%s': %v", expectedCommand, err)
			}

			// They should resolve to the same command
			if cmdViaAlias != cmdViaName {
				t.Errorf("Alias '%s' and command '%s' don't resolve to the same command", alias, expectedCommand)
			}

			// The command name should match expected
			if cmdViaAlias.Name() != expectedCommand {
				t.Errorf("Alias '%s' resolved to '%s', expected '%s'", alias, cmdViaAlias.Name(), expectedCommand)
			}
		})
	}
}

// TestNoAliasConflicts ensures aliases don't conflict with other command names
func TestNoAliasConflicts(t *testing.T) {
	allCommands := make(map[string]bool)
	allAliases := make(map[string]string)

	// Collect all command names
	for _, cmd := range rootCmd.Commands() {
		allCommands[cmd.Name()] = true

		// Collect aliases for this command
		for _, alias := range cmd.Aliases {
			if existingCmd, exists := allAliases[alias]; exists {
				t.Errorf("Alias '%s' is used by both '%s' and '%s'", alias, existingCmd, cmd.Name())
			}
			allAliases[alias] = cmd.Name()

			// Check if alias conflicts with an actual command name
			if allCommands[alias] {
				t.Errorf("Alias '%s' conflicts with existing command name", alias)
			}
		}
	}
}

// TestVersionCommand verifies version command exists
func TestVersionCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("Version command not found: %v", err)
	}

	if cmd == nil {
		t.Fatal("Version command is nil")
	}
}

// TestInitCommand verifies init command exists
func TestInitCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("Init command not found: %v", err)
	}

	if cmd == nil {
		t.Fatal("Init command is nil")
	}

	// Init should not require vault initialization
	if cmd.PersistentPreRunE != nil {
		t.Error("Init command should not have PersistentPreRunE")
	}
}
