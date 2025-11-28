package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// Vault represents the managed storage directory for lx
type Vault struct {
	RootPath      string
	NotesPath     string
	TemplatesPath string
	CachePath     string
	ConfigPath    string
}

// New creates a new Vault instance with XDG-compliant paths
func New() (*Vault, error) {
	rootPath, rootErr := getVaultRoot()
	configPath, configErr := getConfigPath()
	if rootErr != nil {
		return nil, fmt.Errorf("failed to determine vault root: %w", rootErr)
	}
	if configErr != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", configErr)
	}

	vault := &Vault{
		RootPath:      rootPath,
		NotesPath:     filepath.Join(rootPath, "notes"),
		TemplatesPath: filepath.Join(rootPath, "templates"),
		CachePath:     filepath.Join(rootPath, "cache"),
		ConfigPath:    filepath.Join(configPath),
	}

	return vault, nil
}

// getVaultRoot returns the vault root directory path
// Follows XDG Base Directory specification on Unix and uses AppData on Windows
func getVaultRoot() (string, error) {
	// Check XDG_DATA_HOME first (Unix-like systems)
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "lx"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if we're on Windows by looking for APPDATA
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "lx"), nil
	}

	// Fall back to ~/.local/share/lx (Unix-like systems)
	return filepath.Join(homeDir, ".local", "share", "lx"), nil
}

func getConfigPath() (string, error) {
	// Check XDG_CONFIG_HOME first (Unix-like systems)
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "lx", "config.yaml"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if we're on Windows by looking for APPDATA
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "lx-config", "config.yaml"), nil
	}

	// Fall back to ~/.config/lx/config.yaml (Unix-like systems)
	return filepath.Join(homeDir, ".config", "lx", "config.yaml"), nil
}

// Initialize creates the vault directory structure if it doesn't exist
func (v *Vault) Initialize() error {
	directories := []string{
		v.RootPath,
		v.NotesPath,
		v.TemplatesPath,
		v.CachePath,
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Exists checks if the vault has been initialized
func (v *Vault) Exists() bool {
	info, err := os.Stat(v.RootPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetTexInputsEnv returns the TEXINPUTS environment variable value
// This allows LaTeX to find templates in the vault
func (v *Vault) GetTexInputsEnv() string {
	// Format: .:template_path//:
	// . = current directory
	// // = recursive search
	// : = separator
	return fmt.Sprintf(".:%s//:", v.TemplatesPath)
}

// GetNotePath returns the full path for a note file
func (v *Vault) GetNotePath(filename string) string {
	return filepath.Join(v.NotesPath, filename)
}

// GetCachePath returns the full path for a cached file
func (v *Vault) GetCachePath(filename string) string {
	return filepath.Join(v.CachePath, filename)
}

// GetTemplatePath returns the full path for a template file
func (v *Vault) GetTemplatePath(filename string) string {
	return filepath.Join(v.TemplatesPath, filename)
}

// IndexPath returns the path to the graph index file
func (v *Vault) IndexPath() string {
	return filepath.Join(v.CachePath, "index.json")
}

// CleanCache removes all files in the cache directory
func (v *Vault) CleanCache() error {
	entries, err := os.ReadDir(v.CachePath)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(v.CachePath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}
