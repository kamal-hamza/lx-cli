package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

func TestAttachmentService_Store_Success(t *testing.T) {
	// Setup - create a temporary vault
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create a temporary source file
	srcFile := filepath.Join(t.TempDir(), "test-image.png")
	testContent := []byte("fake image content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filename == "" {
		t.Fatal("expected non-empty filename")
	}

	// Verify filename has correct extension
	if filepath.Ext(filename) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(filename))
	}

	// Verify file was created in assets directory
	destPath := v.GetAssetPath(filename)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", destPath)
	}

	// Verify content matches
	storedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	if string(storedContent) != string(testContent) {
		t.Errorf("stored content doesn't match original")
	}
}

func TestAttachmentService_Store_ContentAddressableNaming(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create a source file with known content
	srcFile := filepath.Join(t.TempDir(), "document.pdf")
	testContent := []byte("test pdf content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))
	expectedPrefix := expectedHash[:12]

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify filename starts with hash prefix
	filenameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]
	if filenameWithoutExt != expectedPrefix {
		t.Errorf("expected filename prefix=%s, got %s", expectedPrefix, filenameWithoutExt)
	}
}

func TestAttachmentService_Store_Deduplication(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create a source file
	srcFile := filepath.Join(t.TempDir(), "original.jpg")
	testContent := []byte("identical content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Store the file first time
	filename1, err := svc.Store(context.Background(), srcFile)
	if err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// Create another file with identical content but different name
	srcFile2 := filepath.Join(t.TempDir(), "duplicate.jpg")
	if err := os.WriteFile(srcFile2, testContent, 0644); err != nil {
		t.Fatalf("failed to create second source file: %v", err)
	}

	// Store the duplicate file
	filename2, err := svc.Store(context.Background(), srcFile2)
	if err != nil {
		t.Fatalf("second store failed: %v", err)
	}

	// Assert - both should return the same filename (deduplication)
	if filename1 != filename2 {
		t.Errorf("expected same filename for identical content, got %s and %s", filename1, filename2)
	}

	// Verify only one file exists in assets
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		t.Fatalf("failed to read assets directory: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file in assets (deduplication), got %d", len(files))
	}
}

func TestAttachmentService_Store_DifferentExtensions(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	tests := []struct {
		filename string
		ext      string
	}{
		{"image.png", ".png"},
		{"photo.jpg", ".jpg"},
		{"document.pdf", ".pdf"},
		{"archive.zip", ".zip"},
		{"UPPERCASE.PDF", ".pdf"}, // Should be lowercase
	}

	for _, test := range tests {
		// Create source file
		srcFile := filepath.Join(t.TempDir(), test.filename)
		content := []byte("content for " + test.filename)
		if err := os.WriteFile(srcFile, content, 0644); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		// Execute
		storedFilename, err := svc.Store(context.Background(), srcFile)
		if err != nil {
			t.Fatalf("store failed for %s: %v", test.filename, err)
		}

		// Assert
		if filepath.Ext(storedFilename) != test.ext {
			t.Errorf("file=%s: expected extension=%s, got %s",
				test.filename, test.ext, filepath.Ext(storedFilename))
		}
	}
}

func TestAttachmentService_Store_NonExistentSourceFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Execute with non-existent file
	filename, err := svc.Store(context.Background(), "/non/existent/file.png")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent source file")
	}

	if filename != "" {
		t.Errorf("expected empty filename on error, got %s", filename)
	}
}

func TestAttachmentService_Store_EmptyFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create empty source file
	srcFile := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create empty source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert - should succeed even with empty file
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}

	if filename == "" {
		t.Error("expected non-empty filename even for empty file")
	}

	// Verify empty file was stored
	destPath := v.GetAssetPath(filename)
	storedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	if len(storedContent) != 0 {
		t.Errorf("expected empty stored file, got %d bytes", len(storedContent))
	}
}

func TestAttachmentService_Store_LargeFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create a larger source file (1MB)
	srcFile := filepath.Join(t.TempDir(), "large.bin")
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(srcFile, largeContent, 0644); err != nil {
		t.Fatalf("failed to create large source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify content integrity
	destPath := v.GetAssetPath(filename)
	storedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	if len(storedContent) != len(largeContent) {
		t.Errorf("expected %d bytes, got %d bytes", len(largeContent), len(storedContent))
	}

	// Verify hash matches
	srcHash := sha256.Sum256(largeContent)
	dstHash := sha256.Sum256(storedContent)
	if srcHash != dstHash {
		t.Error("content hash mismatch - file corruption")
	}
}

func TestAttachmentService_Store_FileWithNoExtension(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create source file without extension
	srcFile := filepath.Join(t.TempDir(), "noextension")
	testContent := []byte("content without extension")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no extension or empty extension
	ext := filepath.Ext(filename)
	if ext != "" && ext != "." {
		t.Logf("stored file has extension: %s", ext)
	}

	// Verify file exists
	destPath := v.GetAssetPath(filename)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected file to be stored")
	}
}

func TestAttachmentService_Store_HashLength(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}

	v := &vault.Vault{
		RootPath:   tempDir,
		AssetsPath: assetsDir,
	}
	svc := NewAttachmentService(v)

	// Create source file
	srcFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filename should be: 12-char-hash + extension
	// Remove extension and check hash length
	nameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]
	if len(nameWithoutExt) != 12 {
		t.Errorf("expected 12-character hash prefix, got %d characters: %s",
			len(nameWithoutExt), nameWithoutExt)
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(nameWithoutExt); err != nil {
		t.Errorf("expected valid hex hash prefix, got error: %v", err)
	}
}
