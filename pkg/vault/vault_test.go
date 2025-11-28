package vault

import (
	"path/filepath"
	"testing"
)

func TestVault_GetNotePath(t *testing.T) {
	v := &Vault{
		NotesPath: "/test/vault/notes",
	}

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"simple filename", "note.tex", "/test/vault/notes/note.tex"},
		{"dated filename", "20240101-test.tex", "/test/vault/notes/20240101-test.tex"},
		{"filename with path separators", "subdir/note.tex", "/test/vault/notes/subdir/note.tex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.GetNotePath(tt.filename)
			if result != tt.expected {
				t.Errorf("GetNotePath(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestVault_GetCachePath(t *testing.T) {
	v := &Vault{
		CachePath: "/test/vault/cache",
	}

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"pdf file", "output.pdf", "/test/vault/cache/output.pdf"},
		{"json file", "index.json", "/test/vault/cache/index.json"},
		{"dated pdf", "20240101-note.pdf", "/test/vault/cache/20240101-note.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.GetCachePath(tt.filename)
			if result != tt.expected {
				t.Errorf("GetCachePath(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestVault_GetTemplatePath(t *testing.T) {
	v := &Vault{
		TemplatesPath: "/test/vault/templates",
	}

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"template file", "homework.tex", "/test/vault/templates/homework.tex"},
		{"template with hyphen", "math-common.tex", "/test/vault/templates/math-common.tex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.GetTemplatePath(tt.filename)
			if result != tt.expected {
				t.Errorf("GetTemplatePath(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestVault_GetAssetPath(t *testing.T) {
	v := &Vault{
		AssetsPath: "/test/vault/assets",
	}

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"image file", "abc123def456.png", "/test/vault/assets/abc123def456.png"},
		{"pdf file", "doc123456789.pdf", "/test/vault/assets/doc123456789.pdf"},
		{"no extension", "file123456", "/test/vault/assets/file123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.GetAssetPath(tt.filename)
			if result != tt.expected {
				t.Errorf("GetAssetPath(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestVault_IndexPath(t *testing.T) {
	v := &Vault{
		CachePath: "/test/vault/cache",
	}

	expected := filepath.Join("/test/vault/cache", "index.json")
	result := v.IndexPath()

	if result != expected {
		t.Errorf("IndexPath() = %q, want %q", result, expected)
	}
}

func TestVault_GetTexInputsEnv(t *testing.T) {
	v := &Vault{
		TemplatesPath: "/test/vault/templates",
	}

	result := v.GetTexInputsEnv()
	expected := ".:/test/vault/templates//:"

	if result != expected {
		t.Errorf("GetTexInputsEnv() = %q, want %q", result, expected)
	}

	// Verify format components
	if result[0:2] != ".:" {
		t.Error("TEXINPUTS should start with '.:'")
	}

	if result[len(result)-3:] != "//:" {
		t.Error("TEXINPUTS should end with '//:'")
	}

	// Should contain templates path
	if !contains(result, "/test/vault/templates") {
		t.Error("TEXINPUTS should contain templates path")
	}
}

func TestVault_StructureFields(t *testing.T) {
	v := &Vault{
		RootPath:      "/test/vault",
		NotesPath:     "/test/vault/notes",
		TemplatesPath: "/test/vault/templates",
		AssetsPath:    "/test/vault/assets",
		CachePath:     "/test/vault/cache",
		ConfigPath:    "/test/config/lx/config.yaml",
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"RootPath", v.RootPath, "/test/vault"},
		{"NotesPath", v.NotesPath, "/test/vault/notes"},
		{"TemplatesPath", v.TemplatesPath, "/test/vault/templates"},
		{"AssetsPath", v.AssetsPath, "/test/vault/assets"},
		{"CachePath", v.CachePath, "/test/vault/cache"},
		{"ConfigPath", v.ConfigPath, "/test/config/lx/config.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestVault_PathConsistency(t *testing.T) {
	v := &Vault{
		RootPath:      "/vault",
		NotesPath:     "/vault/notes",
		TemplatesPath: "/vault/templates",
		AssetsPath:    "/vault/assets",
		CachePath:     "/vault/cache",
	}

	// All subdirectories should start with root path
	paths := map[string]string{
		"NotesPath":     v.NotesPath,
		"TemplatesPath": v.TemplatesPath,
		"AssetsPath":    v.AssetsPath,
		"CachePath":     v.CachePath,
	}

	for name, path := range paths {
		if !contains(path, v.RootPath) {
			t.Errorf("%s = %q should contain RootPath %q", name, path, v.RootPath)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
