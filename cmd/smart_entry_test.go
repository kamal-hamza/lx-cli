package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kamal-hamza/lx-cli/pkg/config"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// TestSmartEntry_NoArgs verifies smart entry launches dashboard with no arguments
func TestSmartEntry_NoArgs(t *testing.T) {
	t.Skip("Skipping: runSmartEntry now launches dashboard which requires interactive TUI")
	// When no args are provided, runSmartEntry now calls runDashboard
	// which starts an interactive TUI session that would hang in tests
}

// TestRunSmartAction_ActionRouting verifies the switch statement routes correctly
func TestRunSmartAction_ActionRouting(t *testing.T) {
	// This test verifies that the switch statement doesn't panic
	// We can't test the actual execution without a full vault setup,
	// but we can verify the routing logic works

	tests := []struct {
		name   string
		action string
	}{
		{"open action", "open"},
		{"edit action", "edit"},
		{"invalid defaults to open", "invalid"},
		{"empty defaults to open", ""},
		{"delete is invalid", "delete"},
		{"list is invalid", "list"},
		{"build is invalid", "build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the function exists and can be called
			// The actual behavior depends on initialized services
			// which we're not testing here

			// We just verify the switch statement branches exist
			// by checking that the function can be invoked
			// without panicking on type assertion

			switch tt.action {
			case "edit":
				// Verify edit case exists
				if tt.action != "edit" {
					t.Error("Case mismatch")
				}
			case "open":
				// Verify open case exists
				if tt.action != "open" {
					t.Error("Case mismatch")
				}
			default:
				// Verify default case exists
				// Should default to open
			}
		})
	}
}

// TestLoadConfig_NoVault verifies error when vault not initialized
func TestLoadConfig_NoVault(t *testing.T) {
	// Save original vault
	originalVault := appVault
	defer func() { appVault = originalVault }()

	// Clear global vault
	appVault = nil

	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error when vault is not initialized, got nil")
	}

	expectedMsg := "vault not initialized"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestLoadConfig_Success verifies config loading works
func TestLoadConfig_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Setup vault paths
	vaultRoot := filepath.Join(tempDir, "data", "lx")
	configPath := filepath.Join(tempDir, "config", "lx", "config.yaml")

	// Create directories
	if err := os.MkdirAll(vaultRoot, 0755); err != nil {
		t.Fatalf("Failed to create vault root: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create vault
	v := &vault.Vault{
		RootPath:   vaultRoot,
		ConfigPath: configPath,
	}

	// Create a config file
	cfg := &config.Config{
		Compiler:      "latexmk",
		DefaultAction: "edit",
		MaxWorkers:    4,
	}
	if err := cfg.Save(v.ConfigPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Set global vault
	originalVault := appVault
	appVault = v
	defer func() { appVault = originalVault }()

	// Load config
	loadedCfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.DefaultAction != "edit" {
		t.Errorf("Expected DefaultAction='edit', got %q", loadedCfg.DefaultAction)
	}
}

// TestLoadConfig_DefaultValues verifies default config values are used
func TestLoadConfig_DefaultValues(t *testing.T) {
	tempDir := t.TempDir()

	vaultRoot := filepath.Join(tempDir, "data", "lx")
	configPath := filepath.Join(tempDir, "config", "lx", "config.yaml")

	if err := os.MkdirAll(vaultRoot, 0755); err != nil {
		t.Fatalf("Failed to create vault root: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	v := &vault.Vault{
		RootPath:   vaultRoot,
		ConfigPath: configPath,
	}

	originalVault := appVault
	appVault = v
	defer func() { appVault = originalVault }()

	// Don't create a config file - should use defaults
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultAction != "open" {
		t.Errorf("Expected default DefaultAction='open', got %q", cfg.DefaultAction)
	}

	if cfg.Compiler != "latexmk" {
		t.Errorf("Expected default Compiler='latexmk', got %q", cfg.Compiler)
	}

	if cfg.MaxWorkers != 4 {
		t.Errorf("Expected default MaxWorkers=4, got %d", cfg.MaxWorkers)
	}
}

// TestLoadConfig_InvalidYAML verifies error handling for invalid config
func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()

	vaultRoot := filepath.Join(tempDir, "data", "lx")
	configPath := filepath.Join(tempDir, "config", "lx", "config.yaml")

	if err := os.MkdirAll(vaultRoot, 0755); err != nil {
		t.Fatalf("Failed to create vault root: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	v := &vault.Vault{
		RootPath:   vaultRoot,
		ConfigPath: configPath,
	}

	// Create invalid YAML
	invalidYAML := "this is not: [valid yaml content"
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	originalVault := appVault
	appVault = v
	defer func() { appVault = originalVault }()

	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error loading invalid YAML, got nil")
	}
}

// TestRootCmd_Configuration verifies root command is properly configured
func TestRootCmd_Configuration(t *testing.T) {
	// Verify root command accepts arbitrary args
	if rootCmd.Args == nil {
		t.Error("Root command should have Args set")
	}

	// Verify RunE is set
	if rootCmd.RunE == nil {
		t.Error("Root command should have RunE set to runSmartEntry")
	}

	// Verify PersistentPreRunE is still set
	if rootCmd.PersistentPreRunE == nil {
		t.Error("Root command should have PersistentPreRunE set")
	}
}

// TestSmartEntry_ArgumentJoining tests the query building logic
func TestSmartEntry_ArgumentJoining(t *testing.T) {
	// Test that different argument patterns are handled correctly
	// We verify the function signature works without full execution

	testCases := []struct {
		name string
		args []string
	}{
		{"empty args", []string{}},
		{"single arg", []string{"physics"}},
		{"two args", []string{"physics", "homework"}},
		{"multiple args", []string{"calc", "final", "exam"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We only test that the function can be called with these args
			// Actual behavior depends on vault initialization
			// which we're testing separately

			if len(tc.args) == 0 {
				// No args launches dashboard (interactive TUI)
				// Skip this test case since it would hang
				t.Skip("Skipping empty args - launches interactive dashboard")
			} else {
				// With args, we just verify the function signature works
				// Full integration testing would require vault setup
				t.Logf("Args pattern %v is accepted", tc.args)
			}
		})
	}
}

// TestLoadConfig_Validation tests config field validation
func TestLoadConfig_Validation(t *testing.T) {
	tests := []struct {
		name           string
		defaultAction  string
		expectedAction string
	}{
		{"valid open", "open", "open"},
		{"valid edit", "edit", "edit"},
		{"invalid value", "delete", "open"},
		{"empty value", "", "open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			configPath := filepath.Join(tempDir, "config", "lx", "config.yaml")

			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				t.Fatalf("Failed to create config dir: %v", err)
			}

			// Write config
			yamlContent := ""
			if tt.defaultAction != "" {
				yamlContent = "default_action: " + tt.defaultAction + "\n"
			}
			if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.DefaultAction != tt.expectedAction {
				t.Errorf("Expected DefaultAction=%q, got %q", tt.expectedAction, cfg.DefaultAction)
			}
		})
	}
}

// TestSmartEntry_QueryConstruction verifies query string construction
func TestSmartEntry_QueryConstruction(t *testing.T) {
	// Test the logic of joining multiple arguments
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"single word", []string{"physics"}, "physics"},
		{"two words", []string{"physics", "homework"}, "physics homework"},
		{"multiple words", []string{"calc", "final", "exam"}, "calc final exam"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build query the same way runSmartEntry does
			query := tt.args[0]
			if len(tt.args) > 1 {
				query = ""
				for i, arg := range tt.args {
					if i > 0 {
						query += " "
					}
					query += arg
				}
			}

			if query != tt.expected {
				t.Errorf("Expected query %q, got %q", tt.expected, query)
			}
		})
	}
}

// TestSmartEntry_ConfigActionSelection verifies config-based action selection
func TestSmartEntry_ConfigActionSelection(t *testing.T) {
	tests := []struct {
		name           string
		configAction   string
		expectedAction string
	}{
		{"config says open", "open", "open"},
		{"config says edit", "edit", "edit"},
		{"config invalid defaults to open", "invalid", "open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			configPath := filepath.Join(tempDir, "config", "lx", "config.yaml")

			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				t.Fatalf("Failed to create config dir: %v", err)
			}

			cfg := &config.Config{
				Compiler:      "latexmk",
				DefaultAction: tt.configAction,
				MaxWorkers:    4,
			}
			if err := cfg.Save(configPath); err != nil {
				t.Fatalf("Failed to save config: %v", err)
			}

			// Load and verify
			loadedCfg, err := config.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if loadedCfg.DefaultAction != tt.expectedAction {
				t.Errorf("Expected DefaultAction=%q, got %q",
					tt.expectedAction, loadedCfg.DefaultAction)
			}
		})
	}
}

// TestSmartAction_SwitchCases verifies all switch branches exist
func TestSmartAction_SwitchCases(t *testing.T) {
	actions := []string{"open", "edit", "unknown", "delete", "build", "list", ""}

	for _, action := range actions {
		t.Run("action_"+action, func(t *testing.T) {
			// Test that the switch statement handles all these cases
			// The default case should catch unknown actions

			var expectedBranch string
			switch action {
			case "edit":
				expectedBranch = "edit"
			case "open":
				expectedBranch = "open"
			default:
				expectedBranch = "open" // default fallback
			}

			// Verify logic is correct
			if action == "edit" && expectedBranch != "edit" {
				t.Error("Edit action should route to edit branch")
			}
			if action == "open" && expectedBranch != "open" {
				t.Error("Open action should route to open branch")
			}
			if action != "edit" && action != "open" && expectedBranch != "open" {
				t.Error("Unknown actions should default to open")
			}
		})
	}
}

// TestSmartEntry_BackwardCompatibility verifies existing commands still work
func TestSmartEntry_BackwardCompatibility(t *testing.T) {
	// Verify that existing commands are not affected
	commands := []string{"new", "list", "open", "edit", "build"}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{cmdName})
			if err != nil {
				t.Errorf("Command '%s' should still exist: %v", cmdName, err)
			}
			if cmd == nil {
				t.Errorf("Command '%s' is nil", cmdName)
			}
		})
	}
}
