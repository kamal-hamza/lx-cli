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
	GitAutoPull       bool   `yaml:"git_auto_pull"`
	GitAutoInit       bool   `yaml:"git_auto_init"`
	GitCommitTemplate string `yaml:"git_commit_template"`

	// UI Settings
	DisplayDateFormat  string `yaml:"display_date_format"`
	ColorTheme         string `yaml:"color_theme"`
	SyntaxHighlighting bool   `yaml:"syntax_highlighting"`
	TableWidth         int    `yaml:"table_width"`

	// Search Settings
	GrepCaseSensitive bool `yaml:"grep_case_sensitive"`
	GrepContextLines  int  `yaml:"grep_context_lines"`
	MaxSearchResults  int  `yaml:"max_search_results"`

	// Performance
	WatchDebounceMS        int  `yaml:"watch_debounce_ms"`
	EnableCache            bool `yaml:"enable_cache"`
	CacheExpirationMinutes int  `yaml:"cache_expiration_minutes"`

	// Export
	DefaultExportFormat string `yaml:"default_export_format"`
	ExportIncludeAssets bool   `yaml:"export_include_assets"`

	// Templates
	CustomTemplateDir string `yaml:"custom_template_dir"`
}

// DefaultConfig returns a Config struct with default values
func DefaultConfig() *Config {
	return &Config{
		Compiler:               "latexmk",
		DefaultTemplate:        "",
		DailyTemplate:          "",
		Editor:                 "",
		MaxWorkers:             4,
		LatexmkFlags:           []string{"-pdf", "-interaction=nonstopmode"},
		DefaultAction:          "open",
		DefaultSort:            "date",
		ReverseSort:            false,
		DateFormat:             "20060102",
		PDFViewer:              "",
		Aliases:                make(map[string]string),
		AutoReindex:            true,
		GraphDirection:         "LR",
		GraphMaxNodes:          100,
		AutoBackup:             true,
		BackupRetention:        5,
		GitAutoPull:            true,
		GitAutoInit:            false,
		GitCommitTemplate:      "Auto-sync: {date} {time}",
		DisplayDateFormat:      "2006-01-02",
		ColorTheme:             "auto",
		SyntaxHighlighting:     true,
		TableWidth:             0,
		GrepCaseSensitive:      false,
		GrepContextLines:       2,
		MaxSearchResults:       50,
		WatchDebounceMS:        500,
		EnableCache:            true,
		CacheExpirationMinutes: 30,
		DefaultExportFormat:    "pdf",
		ExportIncludeAssets:    true,
		CustomTemplateDir:      "",
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

	// Apply defaults for essential values if missing
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
		cfg.DateFormat = "20060102"
	}
	if cfg.DisplayDateFormat == "" {
		cfg.DisplayDateFormat = "2006-01-02"
	}
	if cfg.GraphDirection == "" {
		cfg.GraphDirection = "LR"
	}
	if cfg.GitCommitTemplate == "" {
		cfg.GitCommitTemplate = "Auto-sync: {date} {time}"
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
