package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Compiler        string            `yaml:"compiler"`
	DefaultTemplate string            `yaml:"default_template"`
	DailyTemplate   string            `yaml:"daily_template"`
	Editor          string            `yaml:"editor"`
	MaxWorkers      int               `yaml:"max_workers"`
	LatexmkFlags    []string          `yaml:"latexmk_flags"`
	DefaultAction   string            `yaml:"default_action"`
	DefaultSort     string            `yaml:"default_sort"`
	ReverseSort     bool              `yaml:"reverse_sort"`
	DateFormat      string            `yaml:"date_format"`
	PDFViewer       string            `yaml:"pdf_viewer"`
	Aliases         map[string]string `yaml:"aliases"`

	// Feature Flags
	AutoReindex bool `yaml:"auto_reindex"`

	// Graph Settings
	GraphDirection string `yaml:"graph_direction"`
	GraphMaxNodes  int    `yaml:"graph_max_nodes"`

	// Backup Settings
	AutoBackup      bool `yaml:"auto_backup"`
	BackupRetention int  `yaml:"backup_retention"`

	// Git Settings
	GitAutoPull bool `yaml:"git_auto_pull"`
}

// DefaultConfig returns a Config struct with default values
func DefaultConfig() *Config {
	return &Config{
		Compiler:        "latexmk",
		DefaultTemplate: "",
		DailyTemplate:   "",
		Editor:          "",
		MaxWorkers:      4,
		LatexmkFlags:    []string{},
		DefaultAction:   "open",
		DefaultSort:     "date",
		ReverseSort:     false,
		DateFormat:      "2006-01-02",
		PDFViewer:       "xdg-open",
		Aliases:         make(map[string]string),
		AutoReindex:     false,
		GraphDirection:  "LR",
		GraphMaxNodes:   0,
		AutoBackup:      false,
		BackupRetention: 7,
		GitAutoPull:     false,
	}
}

// Load reads configuration from the specified file path
func Load(path string) (*Config, error) {
	// Start with default config
	cfg := DefaultConfig()

	// Try to read the file
	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist, return default config (not an error)
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure map is initialized if nil
	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]string)
	}

	// Apply defaults for missing values
	if cfg.Compiler == "" {
		cfg.Compiler = "latexmk"
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 4
	}
	if cfg.DefaultAction == "" {
		cfg.DefaultAction = "open"
	}
	if cfg.DefaultSort == "" {
		cfg.DefaultSort = "date"
	}
	if cfg.DateFormat == "" {
		cfg.DateFormat = "2006-01-02"
	}
	if cfg.PDFViewer == "" {
		cfg.PDFViewer = "xdg-open"
	}
	if cfg.GraphDirection == "" {
		cfg.GraphDirection = "LR"
	}

	// Validate DefaultAction
	if !isValidDefaultAction(cfg.DefaultAction) {
		cfg.DefaultAction = "open"
	}

	return cfg, nil
}

// Save persists the current configuration to the specified file path
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// isValidDefaultAction checks if the default action is valid
func isValidDefaultAction(action string) bool {
	validActions := []string{"open", "edit", "list", "daily", "graph"}
	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}
