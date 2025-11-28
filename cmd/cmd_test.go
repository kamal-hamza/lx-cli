package cmd

import (
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
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
	createNoteService := services.NewCreateNoteService(mockNoteRepo, mockTemplateRepo)
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
		alias   string
		command string
	}{
		{"list", "list"},
		// Add more aliases if they exist
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.alias})
			if err != nil {
				t.Errorf("Alias '%s' not found: %v", tt.alias, err)
			}
			if cmd == nil {
				t.Errorf("Command for alias '%s' is nil", tt.alias)
			}
		})
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
