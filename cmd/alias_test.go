package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kamal-hamza/lx-cli/pkg/config"
)

func TestValidateAliasName(t *testing.T) {
	tests := []struct {
		name      string
		aliasName string
		wantError bool
	}{
		{
			name:      "valid simple name",
			aliasName: "hw",
			wantError: false,
		},
		{
			name:      "valid with dash",
			aliasName: "quick-note",
			wantError: false,
		},
		{
			name:      "valid with underscore",
			aliasName: "my_alias",
			wantError: false,
		},
		{
			name:      "valid with numbers",
			aliasName: "alias123",
			wantError: false,
		},
		{
			name:      "empty name",
			aliasName: "",
			wantError: true,
		},
		{
			name:      "name with spaces",
			aliasName: "my alias",
			wantError: true,
		},
		{
			name:      "name starting with dash",
			aliasName: "-alias",
			wantError: true,
		},
		{
			name:      "name with special characters",
			aliasName: "alias@test",
			wantError: true,
		},
		{
			name:      "name with slash",
			aliasName: "alias/test",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAliasName(tt.aliasName)
			if tt.wantError && err == nil {
				t.Errorf("validateAliasName(%q) expected error, got nil", tt.aliasName)
			}
			if !tt.wantError && err != nil {
				t.Errorf("validateAliasName(%q) unexpected error: %v", tt.aliasName, err)
			}
		})
	}
}

func TestIsValidAliasChar(t *testing.T) {
	tests := []struct {
		name  string
		char  rune
		valid bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"dash", '-', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"at sign", '@', false},
		{"slash", '/', false},
		{"period", '.', false},
		{"asterisk", '*', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidAliasChar(tt.char)
			if got != tt.valid {
				t.Errorf("isValidAliasChar(%q) = %v, want %v", tt.char, got, tt.valid)
			}
		})
	}
}

func TestIsReservedCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		reserved bool
	}{
		{"reserved: new", "new", true},
		{"reserved: list", "list", true},
		{"reserved: open", "open", true},
		{"reserved: edit", "edit", true},
		{"reserved: alias", "alias", true},
		{"reserved: dashboard", "dashboard", true},
		{"not reserved: hw", "hw", false},
		{"not reserved: mycommand", "mycommand", false},
		{"not reserved: quick", "quick", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReservedCommand(tt.cmdName)
			if got != tt.reserved {
				t.Errorf("isReservedCommand(%q) = %v, want %v", tt.cmdName, got, tt.reserved)
			}
		})
	}
}

func TestExpandAlias(t *testing.T) {
	tests := []struct {
		name     string
		aliasCmd string
		args     []string
		expected []string
	}{
		{
			name:     "simple alias no args",
			aliasCmd: "new -t homework",
			args:     []string{},
			expected: []string{"new", "-t", "homework"},
		},
		{
			name:     "alias with appended args",
			aliasCmd: "new -t homework",
			args:     []string{"assignment1"},
			expected: []string{"new", "-t", "homework", "assignment1"},
		},
		{
			name:     "alias with $1 substitution",
			aliasCmd: "new -t homework -n '$1'",
			args:     []string{"assignment2"},
			expected: []string{"new", "-t", "homework", "-n", "'assignment2'"},
		},
		{
			name:     "alias with multiple substitutions",
			aliasCmd: "new -t $1 -n '$2'",
			args:     []string{"math", "calculus"},
			expected: []string{"new", "-t", "math", "-n", "'calculus'"},
		},
		{
			name:     "alias with $@ substitution",
			aliasCmd: "list -t $@",
			args:     []string{"math", "physics"},
			expected: []string{"list", "-t", "math physics"},
		},
		{
			name:     "single word alias",
			aliasCmd: "list",
			args:     []string{},
			expected: []string{"list"},
		},
		{
			name:     "complex alias",
			aliasCmd: "export -f json -o ~/backup.json",
			args:     []string{},
			expected: []string{"export", "-f", "json", "-o", "~/backup.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandAlias(tt.aliasCmd, tt.args)
			if len(got) != len(tt.expected) {
				t.Errorf("expandAlias() length = %d, want %d\nGot: %v\nWant: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("expandAlias()[%d] = %q, want %q\nFull result: %v",
						i, got[i], tt.expected[i], got)
				}
			}
		})
	}
}

func TestTryResolveAlias(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		cmdName      string
		args         []string
		expectAlias  bool
		expectedArgs []string
	}{
		{
			name: "resolve existing alias",
			cfg: &config.Config{
				Aliases: map[string]string{
					"hw": "new -t homework",
				},
			},
			cmdName:      "hw",
			args:         []string{"assignment1"},
			expectAlias:  true,
			expectedArgs: []string{"new", "-t", "homework", "assignment1"},
		},
		{
			name: "non-existent alias",
			cfg: &config.Config{
				Aliases: map[string]string{
					"hw": "new -t homework",
				},
			},
			cmdName:      "qp",
			args:         []string{},
			expectAlias:  false,
			expectedArgs: nil,
		},
		{
			name:         "nil config",
			cfg:          nil,
			cmdName:      "hw",
			args:         []string{},
			expectAlias:  false,
			expectedArgs: nil,
		},
		{
			name: "nil aliases map",
			cfg: &config.Config{
				Aliases: nil,
			},
			cmdName:      "hw",
			args:         []string{},
			expectAlias:  false,
			expectedArgs: nil,
		},
		{
			name: "alias with variable substitution",
			cfg: &config.Config{
				Aliases: map[string]string{
					"qp": "new -t quick-note -n '$1'",
				},
			},
			cmdName:      "qp",
			args:         []string{"idea"},
			expectAlias:  true,
			expectedArgs: []string{"new", "-t", "quick-note", "-n", "'idea'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isAlias := TryResolveAlias(tt.cfg, tt.cmdName, tt.args)
			if isAlias != tt.expectAlias {
				t.Errorf("TryResolveAlias() isAlias = %v, want %v", isAlias, tt.expectAlias)
			}
			if !tt.expectAlias {
				if got != nil {
					t.Errorf("TryResolveAlias() for non-alias should return nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.expectedArgs) {
				t.Errorf("TryResolveAlias() length = %d, want %d\nGot: %v\nWant: %v",
					len(got), len(tt.expectedArgs), got, tt.expectedArgs)
				return
			}
			for i := range got {
				if got[i] != tt.expectedArgs[i] {
					t.Errorf("TryResolveAlias()[%d] = %q, want %q",
						i, got[i], tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestConfigAliasLoading(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config with aliases
	cfg := &config.Config{
		Compiler:        "latexmk",
		DefaultTemplate: "",
		DailyTemplate:   "",
		DefaultAction:   "open",
		Editor:          "",
		MaxWorkers:      4,
		Aliases: map[string]string{
			"hw":     "new -t homework",
			"today":  "list -s modified -r",
			"backup": "export -f json -o ~/backup.json",
		},
	}

	// Save config
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify aliases were loaded correctly
	if len(loadedCfg.Aliases) != 3 {
		t.Errorf("Expected 3 aliases, got %d", len(loadedCfg.Aliases))
	}

	expectedAliases := map[string]string{
		"hw":     "new -t homework",
		"today":  "list -s modified -r",
		"backup": "export -f json -o ~/backup.json",
	}

	for name, expectedCmd := range expectedAliases {
		gotCmd, ok := loadedCfg.Aliases[name]
		if !ok {
			t.Errorf("Alias %q not found in loaded config", name)
			continue
		}
		if gotCmd != expectedCmd {
			t.Errorf("Alias %q = %q, want %q", name, gotCmd, expectedCmd)
		}
	}
}

func TestConfigAliasEmpty(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config without aliases
	cfg := &config.Config{
		Compiler:        "latexmk",
		DefaultTemplate: "",
		DailyTemplate:   "",
		DefaultAction:   "open",
		Editor:          "",
		MaxWorkers:      4,
	}

	// Save config
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify aliases map is initialized
	if loadedCfg.Aliases == nil {
		t.Error("Aliases map should be initialized, got nil")
	}

	if len(loadedCfg.Aliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(loadedCfg.Aliases))
	}
}

func TestConfigDefaultAliases(t *testing.T) {
	cfg := config.DefaultConfig()

	// Verify aliases map is initialized
	if cfg.Aliases == nil {
		t.Error("Default config should have initialized aliases map")
	}

	if len(cfg.Aliases) != 0 {
		t.Errorf("Default config should have 0 aliases, got %d", len(cfg.Aliases))
	}
}

// TestAliasAddRemoveWorkflow tests a complete add/remove workflow
func TestAliasAddRemoveWorkflow(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	cfg := config.DefaultConfig()
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Add alias
	cfg.Aliases["hw"] = "new -t homework"
	err = cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config with alias: %v", err)
	}

	// Reload and verify
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if cmd, ok := loadedCfg.Aliases["hw"]; !ok {
		t.Error("Alias 'hw' not found after adding")
	} else if cmd != "new -t homework" {
		t.Errorf("Alias 'hw' = %q, want %q", cmd, "new -t homework")
	}

	// Remove alias
	delete(loadedCfg.Aliases, "hw")
	err = loadedCfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config after removing alias: %v", err)
	}

	// Reload and verify removal
	finalCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config after removal: %v", err)
	}

	if _, ok := finalCfg.Aliases["hw"]; ok {
		t.Error("Alias 'hw' should be removed but still exists")
	}
}

// TestAliasComplexCommands tests aliases with complex command structures
func TestAliasComplexCommands(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		command  string
		args     []string
		expected []string
	}{
		{
			name:     "multi-flag alias",
			alias:    "review",
			command:  "list -t review-needed -s modified -r",
			args:     []string{},
			expected: []string{"list", "-t", "review-needed", "-s", "modified", "-r"},
		},
		{
			name:     "alias with path",
			alias:    "backup",
			command:  "export -f json -o ~/Documents/backup.json",
			args:     []string{},
			expected: []string{"export", "-f", "json", "-o", "~/Documents/backup.json"},
		},
		{
			name:     "alias with quoted args",
			alias:    "qp",
			command:  "new -t 'quick note' -n '$1'",
			args:     []string{"idea"},
			expected: []string{"new", "-t", "'quick", "note'", "-n", "'idea'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandAlias(tt.command, tt.args)
			if len(got) != len(tt.expected) {
				t.Errorf("expandAlias() length = %d, want %d\nGot: %v\nWant: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("expandAlias()[%d] = %q, want %q",
						i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// BenchmarkExpandAlias benchmarks alias expansion
func BenchmarkExpandAlias(b *testing.B) {
	aliasCmd := "new -t homework -n '$1'"
	args := []string{"assignment3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expandAlias(aliasCmd, args)
	}
}

// BenchmarkTryResolveAlias benchmarks alias resolution
func BenchmarkTryResolveAlias(b *testing.B) {
	cfg := &config.Config{
		Aliases: map[string]string{
			"hw":     "new -t homework",
			"today":  "list -s modified -r",
			"backup": "export -f json -o ~/backup.json",
			"qp":     "new -t quick-note -e",
			"review": "list -t review-needed",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TryResolveAlias(cfg, "hw", []string{"assignment1"})
	}
}
