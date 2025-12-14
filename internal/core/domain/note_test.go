package domain

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Graph Theory", "graph-theory"},
		{"Complex  Spaces   ", "complex-spaces"},
		{"C++ Programming", "c-programming"}, // simplistic regex behavior
		{"Hello/World", "hello-world"},
		{"My Note!", "my-note"},
	}

	for _, tt := range tests {
		got := GenerateSlug(tt.title)
		if got != tt.expected {
			t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, got, tt.expected)
		}
	}
}

func TestGenerateFilename(t *testing.T) {
	slug := "test-note"
	// Test with standard format
	dateFormat := "20060102"

	filename := GenerateFilename(slug, dateFormat)

	// Expect format: YYYYMMDD-test-note.tex
	datePrefix := time.Now().Format("20060102")
	expected := datePrefix + "-" + slug + ".tex"

	if filename != expected {
		t.Errorf("GenerateFilename(%q, %q) = %q, want %q", slug, dateFormat, filename, expected)
	}

	// Test with hyphenated format
	dateFormat2 := "2006-01-02"
	filename2 := GenerateFilename(slug, dateFormat2)
	datePrefix2 := time.Now().Format("2006-01-02")
	expected2 := datePrefix2 + "-" + slug + ".tex"

	if filename2 != expected2 {
		t.Errorf("GenerateFilename(%q, %q) = %q, want %q", slug, dateFormat2, filename2, expected2)
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		title   string
		isValid bool
	}{
		{"Valid Title", true},
		{"", false},
		{"   ", false},
		{strings.Repeat("a", 201), false},
	}

	for _, tt := range tests {
		err := ValidateTitle(tt.title)
		if (err == nil) != tt.isValid {
			t.Errorf("ValidateTitle(%q) valid = %v, want %v", tt.title, err == nil, tt.isValid)
		}
	}
}

func TestNewNoteHeader(t *testing.T) {
	title := "My Test Note"
	tags := []string{"test", "unit"}
	dateFormat := "20060102"

	header, err := NewNoteHeader(title, tags, dateFormat)
	if err != nil {
		t.Fatalf("NewNoteHeader failed: %v", err)
	}

	if header.Title != title {
		t.Errorf("Title = %q, want %q", header.Title, title)
	}

	expectedSlug := "my-test-note"
	if header.Slug != expectedSlug {
		t.Errorf("Slug = %q, want %q", header.Slug, expectedSlug)
	}

	datePrefix := time.Now().Format("20060102")
	if !strings.HasPrefix(header.Filename, datePrefix) {
		t.Errorf("Filename %q does not start with date %q", header.Filename, datePrefix)
	}

	if len(header.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(header.Tags))
	}
}

func TestNoteHeader_HasTag(t *testing.T) {
	header := &NoteHeader{
		Tags: []string{"Math", "Physics"},
	}

	if !header.HasTag("math") {
		t.Error("HasTag(math) should be true (case insensitive)")
	}
	if !header.HasTag("Physics") {
		t.Error("HasTag(Physics) should be true")
	}
	if header.HasTag("biology") {
		t.Error("HasTag(biology) should be false")
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"20251128-graph-theory.tex", "graph-theory"},
		{"2025-11-28-linear-algebra.tex", "linear-algebra"},
		{"simple-note.tex", "simple-note"},
		{"20251128-nested-slug-name.tex", "nested-slug-name"},
		{"just-a-file", "just-a-file"}, // no extension
	}

	for _, tt := range tests {
		got := ParseFilename(tt.filename)
		if got != tt.expected {
			t.Errorf("ParseFilename(%q) = %q, want %q", tt.filename, got, tt.expected)
		}
	}
}
