package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Compiler string `yaml:"compiler"`

	// Default template for standard notes
	DefaultTemplate string `yaml:"default_template"`

	// Template for daily notes (NEW)
	DailyTemplate string `yaml:"daily_template"`

	Editor     string `yaml:"editor"`
	MaxWorkers int    `yaml:"max_workers"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Compiler:        "latexmk",
		DefaultTemplate: "",
		DailyTemplate:   "", // Default to empty
		Editor:          "",
		MaxWorkers:      4,
	}
}

// Load loads the configuration from the specified path
func Load(path string) (*Config, error) {
	// If file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for any missing values
	if cfg.Compiler == "" {
		cfg.Compiler = "latexmk"
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 4
	}

	return cfg, nil
}

// Save writes the configuration to the specified path
func (c *Config) Save(path string) error {
	// Ensure directory exists
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
