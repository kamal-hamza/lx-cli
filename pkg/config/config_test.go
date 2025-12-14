package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.Compiler != "latexmk" {
		t.Errorf("expected default Compiler='latexmk', got %q", cfg.Compiler)
	}

	if cfg.DefaultTemplate != "" {
		t.Errorf("expected default DefaultTemplate='', got %q", cfg.DefaultTemplate)
	}

	if cfg.DefaultAction != "open" {
		t.Errorf("expected default DefaultAction='open', got %q", cfg.DefaultAction)
	}

	if cfg.Editor != "" {
		t.Errorf("expected default Editor='', got %q", cfg.Editor)
	}

	if cfg.MaxWorkers != 4 {
		t.Errorf("expected default MaxWorkers=4, got %d", cfg.MaxWorkers)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	// Loading a non-existent file should return default config
	cfg, err := Load("/nonexistent/path/config.yaml")

	if err != nil {
		t.Fatalf("unexpected error loading non-existent file: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Should return default values
	if cfg.Compiler != "latexmk" {
		t.Errorf("expected default Compiler='latexmk', got %q", cfg.Compiler)
	}

	if cfg.MaxWorkers != 4 {
		t.Errorf("expected default MaxWorkers=4, got %d", cfg.MaxWorkers)
	}
}

func TestSave_And_Load(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a custom config
	cfg := &Config{
		Compiler:        "tectonic",
		DefaultTemplate: "homework",
		Editor:          "vim",
		MaxWorkers:      8,
	}

	// Save the config
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load the config back
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify values match
	if loadedCfg.Compiler != cfg.Compiler {
		t.Errorf("Compiler: expected %q, got %q", cfg.Compiler, loadedCfg.Compiler)
	}

	if loadedCfg.DefaultTemplate != cfg.DefaultTemplate {
		t.Errorf("DefaultTemplate: expected %q, got %q", cfg.DefaultTemplate, loadedCfg.DefaultTemplate)
	}

	if loadedCfg.Editor != cfg.Editor {
		t.Errorf("Editor: expected %q, got %q", cfg.Editor, loadedCfg.Editor)
	}

	if loadedCfg.MaxWorkers != cfg.MaxWorkers {
		t.Errorf("MaxWorkers: expected %d, got %d", cfg.MaxWorkers, loadedCfg.MaxWorkers)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	// Create a config file with missing values
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a partial config (missing compiler and max_workers)
	yamlContent := `default_template: homework
editor: nvim
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should apply defaults for missing values
	if cfg.Compiler != "latexmk" {
		t.Errorf("expected default Compiler='latexmk', got %q", cfg.Compiler)
	}

	if cfg.MaxWorkers != 4 {
		t.Errorf("expected default MaxWorkers=4, got %d", cfg.MaxWorkers)
	}

	// Should preserve specified values
	if cfg.DefaultTemplate != "homework" {
		t.Errorf("expected DefaultTemplate='homework', got %q", cfg.DefaultTemplate)
	}

	if cfg.Editor != "nvim" {
		t.Errorf("expected Editor='nvim', got %q", cfg.Editor)
	}
}

func TestLoad_EmptyCompiler(t *testing.T) {
	// Create a config file with empty compiler
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `compiler: ""
default_template: homework
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should apply default for empty compiler
	if cfg.Compiler != "latexmk" {
		t.Errorf("expected default Compiler='latexmk' for empty value, got %q", cfg.Compiler)
	}
}

func TestLoad_ZeroMaxWorkers(t *testing.T) {
	// Create a config file with zero max_workers
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `max_workers: 0
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should apply default for zero/negative max_workers
	if cfg.MaxWorkers != 4 {
		t.Errorf("expected default MaxWorkers=4 for zero value, got %d", cfg.MaxWorkers)
	}
}

func TestLoad_NegativeMaxWorkers(t *testing.T) {
	// Create a config file with negative max_workers
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `max_workers: -5
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should apply default for negative max_workers
	if cfg.MaxWorkers != 4 {
		t.Errorf("expected default MaxWorkers=4 for negative value, got %d", cfg.MaxWorkers)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a config file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `compiler: latexmk
default_template: [invalid yaml structure
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected error loading invalid YAML, got nil")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	// Save to a path where directory doesn't exist
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

	cfg := DefaultConfig()
	err := cfg.Save(configPath)

	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("directory was not created")
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}

func TestSave_ValidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	cfg := &Config{
		Compiler:        "tectonic",
		DefaultTemplate: "homework",
		Editor:          "emacs",
		MaxWorkers:      8,
	}

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Read the file and verify it's valid YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	// Verify content contains expected values
	content := string(data)
	if !contains(content, "tectonic") {
		t.Error("config file should contain 'tectonic'")
	}
	if !contains(content, "homework") {
		t.Error("config file should contain 'homework'")
	}
	if !contains(content, "emacs") {
		t.Error("config file should contain 'emacs'")
	}
}

func TestConfig_AllFields(t *testing.T) {
	cfg := &Config{
		Compiler:        "custom-compiler",
		DefaultTemplate: "my-template",
		Editor:          "code",
		MaxWorkers:      16,
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Compiler", cfg.Compiler, "custom-compiler"},
		{"DefaultTemplate", cfg.DefaultTemplate, "my-template"},
		{"Editor", cfg.Editor, "code"},
		{"MaxWorkers", cfg.MaxWorkers, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestDefaultAction_DefaultValue(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultAction != "open" {
		t.Errorf("expected default DefaultAction='open', got %q", cfg.DefaultAction)
	}
}

func TestDefaultAction_ValidValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"open", "open", "open"},
		{"edit", "edit", "edit"},
		{"empty defaults to open", "", "open"},
		{"invalid defaults to open", "invalid", "open"},
		{"delete is invalid", "delete", "open"},
		{"list is invalid", "list", "open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			yamlContent := ""
			if tt.value != "" {
				yamlContent = "default_action: " + tt.value + "\n"
			}

			if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
				t.Fatalf("failed to create test config file: %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.DefaultAction != tt.expected {
				t.Errorf("DefaultAction: expected %q, got %q", tt.expected, cfg.DefaultAction)
			}
		})
	}
}

func TestDefaultAction_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	tests := []struct {
		name  string
		value string
	}{
		{"open", "open"},
		{"edit", "edit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Compiler:      "latexmk",
				DefaultAction: tt.value,
				MaxWorkers:    4,
			}

			err := cfg.Save(configPath)
			if err != nil {
				t.Fatalf("failed to save config: %v", err)
			}

			loadedCfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if loadedCfg.DefaultAction != tt.value {
				t.Errorf("DefaultAction: expected %q, got %q", tt.value, loadedCfg.DefaultAction)
			}
		})
	}
}

func TestLoad_PreservesDefaultAction(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `compiler: latexmk
default_action: edit
editor: vim
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.DefaultAction != "edit" {
		t.Errorf("expected DefaultAction='edit', got %q", cfg.DefaultAction)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
