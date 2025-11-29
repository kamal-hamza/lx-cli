package services

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
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
	// Use mock repository
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create a temporary source file
	srcFile := filepath.Join(t.TempDir(), "test-image.png")
	testContent := []byte("fake image content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute with explicit name and description
	filename, err := svc.Store(context.Background(), srcFile, "test-image", "description")

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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create a source file with known content
	srcFile := filepath.Join(t.TempDir(), "document.pdf")
	testContent := []byte("test pdf content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Calculate expected hash logic if we were using hash-based naming,
	// but now we use explicit naming "document".
	// The test should verify it uses the name provided.

	// Execute
	filename, err := svc.Store(context.Background(), srcFile, "document", "desc")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify filename is document.pdf
	if filename != "document.pdf" {
		t.Errorf("expected filename=document.pdf, got %s", filename)
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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create a source file
	srcFile := filepath.Join(t.TempDir(), "original.jpg")
	testContent := []byte("identical content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Store the file first time
	filename1, err := svc.Store(context.Background(), srcFile, "graph", "first upload")
	if err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// Create another file with identical content but different name on disk
	srcFile2 := filepath.Join(t.TempDir(), "duplicate.jpg")
	if err := os.WriteFile(srcFile2, testContent, 0644); err != nil {
		t.Fatalf("failed to create second source file: %v", err)
	}

	// Store the duplicate file with SAME target name "graph"
	filename2, err := svc.Store(context.Background(), srcFile2, "graph", "second upload")
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

	// Ignore .manifest.json if present (it's not created by file system mock here but logic might)
	count := 0
	for _, f := range files {
		if f.Name() != ".manifest.json" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected 1 file in assets (deduplication), got %d", count)
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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

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
		nameWithoutExt := "asset"
		storedFilename, err := svc.Store(context.Background(), srcFile, nameWithoutExt, "desc")
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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Execute with non-existent file
	filename, err := svc.Store(context.Background(), "/non/existent/file.png", "fail", "desc")

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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create empty source file
	srcFile := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create empty source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile, "empty", "desc")

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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

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
	filename, err := svc.Store(context.Background(), srcFile, "large", "desc")

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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create source file without extension
	srcFile := filepath.Join(t.TempDir(), "noextension")
	testContent := []byte("content without extension")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute
	filename, err := svc.Store(context.Background(), srcFile, "noext", "desc")

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

func TestAttachmentService_Store_CollisionResolution(t *testing.T) {
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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// File 1
	srcFile1 := filepath.Join(t.TempDir(), "a.png")
	os.WriteFile(srcFile1, []byte("content A"), 0644)

	// File 2 (Different content)
	srcFile2 := filepath.Join(t.TempDir(), "b.png")
	os.WriteFile(srcFile2, []byte("content B"), 0644)

	// Store first
	name1, _ := svc.Store(context.Background(), srcFile1, "graph", "desc A")
	if name1 != "graph.png" {
		t.Errorf("expected graph.png, got %s", name1)
	}

	// Store second (Name collision -> should auto-rename)
	name2, _ := svc.Store(context.Background(), srcFile2, "graph", "desc B")

	// Should be graph-1.png
	if name2 != "graph-1.png" {
		t.Errorf("expected graph-1.png for collision, got %s", name2)
	}
}
