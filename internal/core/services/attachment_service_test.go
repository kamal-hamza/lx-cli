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
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// Create a temporary source file
	srcFile := filepath.Join(t.TempDir(), "test-image.png")
	testContent := []byte("fake image content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Execute
	filename, isDuplicate, err := svc.Store(context.Background(), srcFile, "test-image", "description")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if isDuplicate {
		t.Error("expected new file, got duplicate")
	}

	if filename == "" {
		t.Fatal("expected non-empty filename")
	}

	// Verify filename has correct extension
	if filepath.Ext(filename) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(filename))
	}

	// Verify content matches
	destPath := v.GetAssetPath(filename)
	storedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	if string(storedContent) != string(testContent) {
		t.Errorf("stored content doesn't match original")
	}
}

func TestAttachmentService_Store_GlobalDeduplication(t *testing.T) {
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

	// 1. Create a source file
	content := []byte("identical content")
	srcFile1 := filepath.Join(t.TempDir(), "original.jpg")
	if err := os.WriteFile(srcFile1, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// 2. Store the file first time
	filename1, _, err := svc.Store(context.Background(), srcFile1, "graph", "first upload")
	if err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// 3. Create another file with IDENTICAL content but DIFFERENT name on disk
	srcFile2 := filepath.Join(t.TempDir(), "duplicate-download.jpg")
	if err := os.WriteFile(srcFile2, content, 0644); err != nil {
		t.Fatalf("failed to create second source file: %v", err)
	}

	// 4. Store the duplicate file with a COMPLETELY NEW name "chart"
	// Old behavior: would create chart.jpg
	// New behavior: should detect hash match and return graph.jpg
	filename2, isDuplicate, err := svc.Store(context.Background(), srcFile2, "chart", "second upload")
	if err != nil {
		t.Fatalf("second store failed: %v", err)
	}

	// Assert
	if !isDuplicate {
		t.Error("expected duplicate to be detected")
	}

	if filename1 != filename2 {
		t.Errorf("expected global deduplication (filename %s), but got new file %s", filename1, filename2)
	}
}

func TestAttachmentService_Store_NonExistentSourceFile(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	os.MkdirAll(assetsDir, 0755)

	v := &vault.Vault{RootPath: tempDir, AssetsPath: assetsDir}
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	filename, _, err := svc.Store(context.Background(), "/non/existent/file.png", "fail", "desc")

	if err == nil {
		t.Fatal("expected error for non-existent source file")
	}
	if filename != "" {
		t.Errorf("expected empty filename on error, got %s", filename)
	}
}

func TestAttachmentService_Store_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	os.MkdirAll(assetsDir, 0755)

	v := &vault.Vault{RootPath: tempDir, AssetsPath: assetsDir}
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	srcFile := filepath.Join(t.TempDir(), "empty.txt")
	os.WriteFile(srcFile, []byte{}, 0644)

	filename, _, err := svc.Store(context.Background(), srcFile, "empty", "desc")

	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if filename == "" {
		t.Error("expected non-empty filename")
	}
}

func TestAttachmentService_Store_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	os.MkdirAll(assetsDir, 0755)

	v := &vault.Vault{RootPath: tempDir, AssetsPath: assetsDir}
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	srcFile := filepath.Join(t.TempDir(), "large.bin")
	largeContent := make([]byte, 1024*1024) // 1MB
	os.WriteFile(srcFile, largeContent, 0644)

	filename, _, err := svc.Store(context.Background(), srcFile, "large", "desc")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	destPath := v.GetAssetPath(filename)
	storedContent, _ := os.ReadFile(destPath)

	// Verify hash matches
	srcHash := sha256.Sum256(largeContent)
	dstHash := sha256.Sum256(storedContent)
	if srcHash != dstHash {
		t.Error("content hash mismatch")
	}
}

func TestAttachmentService_Store_CollisionResolution(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	os.MkdirAll(assetsDir, 0755)

	v := &vault.Vault{RootPath: tempDir, AssetsPath: assetsDir}
	mockRepo := mocks.NewMockAssetRepository()
	svc := NewAttachmentService(v, mockRepo)

	// File 1
	srcFile1 := filepath.Join(t.TempDir(), "a.png")
	os.WriteFile(srcFile1, []byte("content A"), 0644)

	// File 2 (Different content)
	srcFile2 := filepath.Join(t.TempDir(), "b.png")
	os.WriteFile(srcFile2, []byte("content B"), 0644)

	// Store first
	name1, _, _ := svc.Store(context.Background(), srcFile1, "graph", "desc A")
	if name1 != "graph.png" {
		t.Errorf("expected graph.png, got %s", name1)
	}

	// Store second (Name collision -> should auto-rename)
	name2, _, _ := svc.Store(context.Background(), srcFile2, "graph", "desc B")

	// Should be graph-1.png
	if name2 != "graph-1.png" {
		t.Errorf("expected graph-1.png for collision, got %s", name2)
	}
}
