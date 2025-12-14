package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGrepService_ScanFile_EmptyQuery(t *testing.T) {
	// Setup: caseSensitive=false, maxResults=0 (unlimited)
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-test-note.tex")
	content := `Line 1: Introduction
Line 2: Details
Line 3: Conclusion`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute with empty query (should return all non-empty lines)
	matches := svc.scanFile(testFile, "", true)

	// Assert
	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d", len(matches))
	}

	// Verify slug extraction from filename
	expectedSlug := "test-note"
	if len(matches) > 0 && matches[0].Slug != expectedSlug {
		t.Errorf("expected slug=%s, got %s", expectedSlug, matches[0].Slug)
	}
}

func TestGrepService_ScanFile_WithQuery(t *testing.T) {
	// Setup: caseSensitive=false, maxResults=0
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-search-test.tex")
	content := `This line contains the word topology
This line does not
Another line with TOPOLOGY in caps
Final line without the word`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute with query
	matches := svc.scanFile(testFile, "topology", false)

	// Assert - should find 2 lines (case-insensitive)
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	// Verify line numbers are correct
	if len(matches) >= 2 {
		if matches[0].LineNum != 1 {
			t.Errorf("first match: expected line 1, got %d", matches[0].LineNum)
		}
		if matches[1].LineNum != 3 {
			t.Errorf("second match: expected line 3, got %d", matches[1].LineNum)
		}
	}
}

func TestGrepService_ScanFile_SkipsEmptyLines(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file with empty lines
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-test.tex")
	content := `Line 1

Line 3

Line 5`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute with empty query (searchAll=true)
	matches := svc.scanFile(testFile, "", true)

	// Assert - should skip empty and whitespace-only lines
	if len(matches) != 3 {
		t.Errorf("expected 3 non-empty matches, got %d", len(matches))
	}

	// Verify content
	expectedContents := []string{"Line 1", "Line 3", "Line 5"}
	for i, match := range matches {
		if i >= len(expectedContents) {
			break
		}
		if match.Content != expectedContents[i] {
			t.Errorf("match[%d]: expected content=%s, got %s", i, expectedContents[i], match.Content)
		}
	}
}

func TestGrepService_ScanFile_SlugExtraction(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	tests := []struct {
		filename     string
		expectedSlug string
	}{
		{"20240101-topology.tex", "topology"},
		{"20231225-graph-theory.tex", "graph-theory"},
		{"simple.tex", "simple"},
		{"20240101-multi-word-slug.tex", "multi-word-slug"},
	}

	for _, test := range tests {
		// Create temporary test file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, test.filename)
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Execute
		matches := svc.scanFile(testFile, "", true)

		// Assert
		if len(matches) == 0 {
			t.Fatalf("expected at least 1 match for %s", test.filename)
		}

		if matches[0].Slug != test.expectedSlug {
			t.Errorf("filename=%s: expected slug=%s, got %s",
				test.filename, test.expectedSlug, matches[0].Slug)
		}

		if matches[0].Filename != test.filename {
			t.Errorf("expected filename=%s, got %s", test.filename, matches[0].Filename)
		}
	}
}

func TestGrepService_ScanFile_CaseInsensitiveSearch(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-test.tex")
	content := `lowercase topology
UPPERCASE TOPOLOGY
MiXeD CaSe ToPoLoGy`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute with lowercase query
	matches := svc.scanFile(testFile, "topology", false)

	// Assert - all 3 lines should match (case-insensitive)
	if len(matches) != 3 {
		t.Errorf("expected 3 matches (case-insensitive), got %d", len(matches))
	}
}

func TestGrepService_ScanFile_NonExistentFile(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Execute with non-existent file
	matches := svc.scanFile("/non/existent/file.tex", "query", false)

	// Assert - should return nil or empty slice
	if matches != nil {
		t.Errorf("expected nil or empty matches for non-existent file, got %d matches", len(matches))
	}
}

func TestGrepService_ScanFile_ExactLineContent(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-test.tex")
	content := `\section{Introduction}
This is a paragraph with topology.
\subsection{Details}
More content here.`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute
	matches := svc.scanFile(testFile, "topology", false)

	// Assert
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	expectedContent := "This is a paragraph with topology."
	if matches[0].Content != expectedContent {
		t.Errorf("expected content=%s, got %s", expectedContent, matches[0].Content)
	}

	if matches[0].LineNum != 2 {
		t.Errorf("expected line number=2, got %d", matches[0].LineNum)
	}
}

func TestGrepService_ScanFile_MultipleOccurrencesInLine(t *testing.T) {
	// Setup
	svc := NewGrepService("/tmp/test-vault", false, 0)

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "20240101-test.tex")
	content := `topology and more topology in topology`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute
	matches := svc.scanFile(testFile, "topology", false)

	// Assert - should return the line once even though query appears 3 times
	if len(matches) != 1 {
		t.Errorf("expected 1 match (line should appear once), got %d", len(matches))
	}
}

func TestGrepService_Execute_Integration(t *testing.T) {
	// Setup - create a temporary vault structure
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("failed to create notes directory: %v", err)
	}

	// Create test notes
	notes := []struct {
		filename string
		content  string
	}{
		{"20240101-topology.tex", "Introduction to topology concepts\nBasic definitions"},
		{"20240102-algebra.tex", "Linear algebra and topology\nMatrix operations"},
		{"20240103-calculus.tex", "Derivatives and integrals\nNo related concepts"},
	}

	for _, note := range notes {
		path := filepath.Join(notesDir, note.filename)
		if err := os.WriteFile(path, []byte(note.content), 0644); err != nil {
			t.Fatalf("failed to create note file: %v", err)
		}
	}

	// Execute
	svc := NewGrepService(tempDir, false, 0)
	matches, err := svc.Execute(context.Background(), "topology")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 2 lines across 2 files containing "topology"
	if len(matches) != 2 {
		t.Errorf("expected 2 matches across files, got %d", len(matches))
	}

	// Verify slugs
	slugCounts := make(map[string]int)
	for _, match := range matches {
		slugCounts[match.Slug]++
	}

	if slugCounts["topology"] != 1 {
		t.Errorf("expected 1 match in topology note, got %d", slugCounts["topology"])
	}

	if slugCounts["algebra"] != 1 {
		t.Errorf("expected 1 match in algebra note, got %d", slugCounts["algebra"])
	}

	if slugCounts["calculus"] != 0 {
		t.Errorf("expected 0 matches in calculus note, got %d", slugCounts["calculus"])
	}
}

func TestGrepService_Execute_EmptyQuery(t *testing.T) {
	// Setup - create a temporary vault structure
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("failed to create notes directory: %v", err)
	}

	// Create a test note
	notePath := filepath.Join(notesDir, "20240101-test.tex")
	content := "Line 1\nLine 2\nLine 3"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create note file: %v", err)
	}

	// Execute with empty query
	svc := NewGrepService(tempDir, false, 0)
	matches, err := svc.Execute(context.Background(), "")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return all non-empty lines
	if len(matches) != 3 {
		t.Errorf("expected 3 matches for empty query, got %d", len(matches))
	}
}

func TestGrepService_Execute_NoNotesDirectory(t *testing.T) {
	// Setup - use a directory without a notes subdirectory
	tempDir := t.TempDir()

	// Execute
	svc := NewGrepService(tempDir, false, 0)
	matches, err := svc.Execute(context.Background(), "test")

	// Assert - should return error
	if err == nil {
		t.Error("expected error for missing notes directory")
	}

	if matches != nil {
		t.Errorf("expected nil matches on error, got %d matches", len(matches))
	}
}

func TestGrepService_Execute_EmptyNotesDirectory(t *testing.T) {
	// Setup - create empty notes directory
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("failed to create notes directory: %v", err)
	}

	// Execute
	svc := NewGrepService(tempDir, false, 0)
	matches, err := svc.Execute(context.Background(), "test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("expected 0 matches in empty directory, got %d", len(matches))
	}
}

func TestGrepService_Execute_OnlyTexFiles(t *testing.T) {
	// Setup - create notes directory with mixed file types
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("failed to create notes directory: %v", err)
	}

	// Create various files
	files := []struct {
		name    string
		content string
	}{
		{"note1.tex", "test content"},
		{"note2.txt", "test content"},
		{"note3.tex", "test content"},
		{"readme.md", "test content"},
	}

	for _, file := range files {
		path := filepath.Join(notesDir, file.name)
		if err := os.WriteFile(path, []byte(file.content), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Execute
	svc := NewGrepService(tempDir, false, 0)
	matches, err := svc.Execute(context.Background(), "test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only search .tex files (2 matches)
	if len(matches) != 2 {
		t.Errorf("expected 2 matches from .tex files only, got %d", len(matches))
	}
}
