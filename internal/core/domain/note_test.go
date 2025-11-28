package domain

import (
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple title",
			title:    "Graph Theory",
			expected: "graph-theory",
		},
		{
			name:     "title with multiple spaces",
			title:    "Linear Algebra  Notes",
			expected: "linear-algebra-notes",
		},
		{
			name:     "title with special characters",
			title:    "Chemistry Lab #1",
			expected: "chemistry-lab-1",
		},
		{
			name:     "title with punctuation",
			title:    "Calculus: Chapter 3",
			expected: "calculus-chapter-3",
		},
		{
			name:     "title with mixed case",
			title:    "Machine Learning Basics",
			expected: "machine-learning-basics",
		},
		{
			name:     "title with multiple special chars",
			title:    "C++ Programming & Design",
			expected: "c-programming-design",
		},
		{
			name:     "title with trailing spaces",
			title:    "  Physics Notes  ",
			expected: "physics-notes",
		},
		{
			name:     "title with unicode",
			title:    "Matemáticas Avanzadas",
			expected: "matem-ticas-avanzadas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	slug := "test-note"
	filename := GenerateFilename(slug)

	// Should end with -test-note.tex
	expectedSuffix := "-test-note.tex"
	if len(filename) < len(expectedSuffix) {
		t.Errorf("GenerateFilename(%q) = %q, too short", slug, filename)
		return
	}

	actualSuffix := filename[len(filename)-len(expectedSuffix):]
	if actualSuffix != expectedSuffix {
		t.Errorf("GenerateFilename(%q) = %q, should end with %q", slug, filename, expectedSuffix)
	}

	// Should start with 8 digit date (YYYYMMDD)
	if len(filename) < 8 {
		t.Errorf("GenerateFilename(%q) = %q, should start with date", slug, filename)
		return
	}

	datePrefix := filename[:8]
	for _, c := range datePrefix {
		if c < '0' || c > '9' {
			t.Errorf("GenerateFilename(%q) = %q, date prefix %q should be numeric", slug, filename, datePrefix)
			break
		}
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "standard filename",
			filename: "20251127-graph-theory.tex",
			expected: "graph-theory",
		},
		{
			name:     "filename with multiple hyphens",
			filename: "20251127-linear-algebra-notes.tex",
			expected: "linear-algebra-notes",
		},
		{
			name:     "filename without date",
			filename: "notes.tex",
			expected: "notes",
		},
		{
			name:     "filename without extension",
			filename: "20251127-chemistry",
			expected: "chemistry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("ParseFilename(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "valid title",
			title:   "Graph Theory Notes",
			wantErr: false,
		},
		{
			name:    "empty title",
			title:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only title",
			title:   "   ",
			wantErr: true,
		},
		{
			name:    "very long title",
			title:   string(make([]byte, 201)),
			wantErr: true,
		},
		{
			name:    "title at max length",
			title:   string(make([]byte, 200)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid template name",
			template: "homework",
			wantErr:  false,
		},
		{
			name:     "template with hyphen",
			template: "math-common",
			wantErr:  false,
		},
		{
			name:     "template with underscore",
			template: "hw_template",
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			wantErr:  true,
		},
		{
			name:     "template with spaces",
			template: "my template",
			wantErr:  true,
		},
		{
			name:     "template with special chars",
			template: "template!@#",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate(%q) error = %v, wantErr %v", tt.template, err, tt.wantErr)
			}
		})
	}
}

func TestNewNoteHeader(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		tags    []string
		wantErr bool
	}{
		{
			name:    "valid note",
			title:   "Test Note",
			tags:    []string{"test", "example"},
			wantErr: false,
		},
		{
			name:    "note without tags",
			title:   "Test Note",
			tags:    nil,
			wantErr: false,
		},
		{
			name:    "invalid title",
			title:   "",
			tags:    []string{"test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := NewNoteHeader(tt.title, tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNoteHeader(%q, %v) error = %v, wantErr %v", tt.title, tt.tags, err, tt.wantErr)
				return
			}

			if err == nil {
				if header.Title != tt.title {
					t.Errorf("NewNoteHeader().Title = %q, want %q", header.Title, tt.title)
				}
				if header.Slug == "" {
					t.Errorf("NewNoteHeader().Slug should not be empty")
				}
				if header.Filename == "" {
					t.Errorf("NewNoteHeader().Filename should not be empty")
				}
				if header.Date == "" {
					t.Errorf("NewNoteHeader().Date should not be empty")
				}
				if tt.tags == nil && len(header.Tags) != 0 {
					t.Errorf("NewNoteHeader().Tags = %v, want empty slice", header.Tags)
				}
			}
		})
	}
}

func TestNoteHeaderHasTag(t *testing.T) {
	header := &NoteHeader{
		Tags: []string{"math", "algebra", "homework"},
	}

	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "exact match",
			tag:      "math",
			expected: true,
		},
		{
			name:     "case insensitive match",
			tag:      "MATH",
			expected: true,
		},
		{
			name:     "no match",
			tag:      "science",
			expected: false,
		},
		{
			name:     "partial match should not work",
			tag:      "mat",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := header.HasTag(tt.tag)
			if result != tt.expected {
				t.Errorf("HasTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestNoteHeaderGetTagsString(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "multiple tags",
			tags:     []string{"math", "algebra"},
			expected: "math, algebra",
		},
		{
			name:     "single tag",
			tags:     []string{"science"},
			expected: "science",
		},
		{
			name:     "no tags",
			tags:     []string{},
			expected: "-",
		},
		{
			name:     "nil tags",
			tags:     nil,
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &NoteHeader{Tags: tt.tags}
			result := header.GetTagsString()
			if result != tt.expected {
				t.Errorf("GetTagsString() = %q, want %q", result, tt.expected)
			}
		})
	}
}
