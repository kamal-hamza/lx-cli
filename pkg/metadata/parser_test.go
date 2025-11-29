package metadata

import (
	"strings"
	"testing"
	"time"
)

func TestParser_Parse_ValidMetadata(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected Metadata
	}{
		{
			name: "basic metadata",
			content: `%% Metadata
%% title: Test Note
%% date: 2024-01-15
%% tags: math, physics

\documentclass{article}`,
			expected: Metadata{
				Title: "Test Note",
				Date:  "2024-01-15",
				Tags:  []string{"math", "physics"},
			},
		},
		{
			name: "metadata with no tags",
			content: `%% Metadata
%% title: Simple Note
%% date: 2024-01-15
%% tags:

\documentclass{article}`,
			expected: Metadata{
				Title: "Simple Note",
				Date:  "2024-01-15",
				Tags:  []string{},
			},
		},
		{
			name: "metadata with single tag",
			content: `%% Metadata
%% title: Physics Notes
%% date: 2024-01-15
%% tags: physics

\documentclass{article}`,
			expected: Metadata{
				Title: "Physics Notes",
				Date:  "2024-01-15",
				Tags:  []string{"physics"},
			},
		},
		{
			name: "metadata with single % signs",
			content: `% Metadata
% title: Old Format Note
% date: 2024-01-15
% tags: legacy

\documentclass{article}`,
			expected: Metadata{
				Title: "Old Format Note",
				Date:  "2024-01-15",
				Tags:  []string{"legacy"},
			},
		},
		{
			name: "metadata with extra whitespace",
			content: `%%   Metadata
%%   title:   Whitespace Test
%%   date:   2024-01-15
%%   tags:   tag1  ,  tag2  ,  tag3

\documentclass{article}`,
			expected: Metadata{
				Title: "Whitespace Test",
				Date:  "2024-01-15",
				Tags:  []string{"tag1", "tag2", "tag3"},
			},
		},
		{
			name: "metadata with special characters in title",
			content: `%% Metadata
%% title: Math: Calculus & Analysis (Part 1)
%% date: 2024-01-15
%% tags: math

\documentclass{article}`,
			expected: Metadata{
				Title: "Math: Calculus & Analysis (Part 1)",
				Date:  "2024-01-15",
				Tags:  []string{"math"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(false)
			result, err := parser.Parse(tt.content)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Metadata.Title != tt.expected.Title {
				t.Errorf("Title: expected %q, got %q", tt.expected.Title, result.Metadata.Title)
			}

			if result.Metadata.Date != tt.expected.Date {
				t.Errorf("Date: expected %q, got %q", tt.expected.Date, result.Metadata.Date)
			}

			if len(result.Metadata.Tags) != len(tt.expected.Tags) {
				t.Errorf("Tags count: expected %d, got %d", len(tt.expected.Tags), len(result.Metadata.Tags))
			} else {
				for i, tag := range tt.expected.Tags {
					if result.Metadata.Tags[i] != tag {
						t.Errorf("Tag[%d]: expected %q, got %q", i, tag, result.Metadata.Tags[i])
					}
				}
			}
		})
	}
}

func TestParser_Parse_MissingMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		strict  bool
		wantErr bool
	}{
		{
			name: "no metadata block - non-strict",
			content: `\documentclass{article}
\begin{document}
Content here
\end{document}`,
			strict:  false,
			wantErr: false,
		},
		{
			name: "no metadata block - strict",
			content: `\documentclass{article}
\begin{document}
Content here
\end{document}`,
			strict:  true,
			wantErr: true,
		},
		{
			name: "empty metadata block - non-strict",
			content: `%% Metadata

\documentclass{article}`,
			strict:  false,
			wantErr: false,
		},
		{
			name: "empty metadata block - strict",
			content: `%% Metadata

\documentclass{article}`,
			strict:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.strict)
			_, err := parser.Parse(tt.content)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParser_Parse_InvalidDate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		strict  bool
		wantErr bool
	}{
		{
			name: "invalid date format - non-strict",
			content: `%% Metadata
%% title: Test
%% date: 01/15/2024
%% tags: test`,
			strict:  false,
			wantErr: false, // Non-strict should succeed but add error to result
		},
		{
			name: "invalid date format - strict",
			content: `%% Metadata
%% title: Test
%% date: 01/15/2024
%% tags: test`,
			strict:  true,
			wantErr: true,
		},
		{
			name: "malformed date - non-strict",
			content: `%% Metadata
%% title: Test
%% date: not-a-date
%% tags: test`,
			strict:  false,
			wantErr: false,
		},
		{
			name: "malformed date - strict",
			content: `%% Metadata
%% title: Test
%% date: not-a-date
%% tags: test`,
			strict:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.strict)
			result, err := parser.Parse(tt.content)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Non-strict mode should still populate date even if invalid
			if !tt.strict && result != nil && result.Metadata.Date == "" {
				t.Error("Expected date to be populated in non-strict mode")
			}
		})
	}
}

func TestParser_Parse_DuplicateFields(t *testing.T) {
	content := `%% Metadata
%% title: First Title
%% title: Second Title
%% date: 2024-01-15
%% tags: tag1
%% tags: tag2

\documentclass{article}`

	parser := NewParser(false)
	result, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use first title
	if result.Metadata.Title != "First Title" {
		t.Errorf("Expected first title, got %q", result.Metadata.Title)
	}

	// Should have warnings about duplicates
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings about duplicate fields")
	}

	// Tags should be merged
	if len(result.Metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags (merged), got %d", len(result.Metadata.Tags))
	}
}

func TestParser_Parse_DuplicateTags(t *testing.T) {
	content := `%% Metadata
%% title: Test
%% date: 2024-01-15
%% tags: math, physics, math, calculus, Physics

\documentclass{article}`

	parser := NewParser(false)
	result, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should remove duplicate tags (case-insensitive)
	expectedUnique := 3 // math, physics, calculus
	if len(result.Metadata.Tags) != expectedUnique {
		t.Errorf("Expected %d unique tags, got %d: %v", expectedUnique, len(result.Metadata.Tags), result.Metadata.Tags)
	}
}

func TestParser_Parse_UnknownFields(t *testing.T) {
	content := `%% Metadata
%% title: Test
%% date: 2024-01-15
%% author: John Doe
%% category: science
%% tags: test

\documentclass{article}`

	parser := NewParser(false)
	result, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have warnings about unknown fields
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings about unknown fields")
	}

	// Should still parse known fields
	if result.Metadata.Title != "Test" {
		t.Errorf("Expected title 'Test', got %q", result.Metadata.Title)
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		contains []string
	}{
		{
			name: "complete metadata",
			metadata: Metadata{
				Title: "Test Note",
				Date:  "2024-01-15",
				Tags:  []string{"math", "physics"},
			},
			contains: []string{
				"%% Metadata",
				"%% title: Test Note",
				"%% date: 2024-01-15",
				"%% tags: math, physics",
			},
		},
		{
			name: "metadata without tags",
			metadata: Metadata{
				Title: "Simple Note",
				Date:  "2024-01-15",
				Tags:  []string{},
			},
			contains: []string{
				"%% Metadata",
				"%% title: Simple Note",
				"%% date: 2024-01-15",
				"%% tags: ",
			},
		},
		{
			name: "metadata with empty date",
			metadata: Metadata{
				Title: "No Date Note",
				Date:  "",
				Tags:  []string{"test"},
			},
			contains: []string{
				"%% Metadata",
				"%% title: No Date Note",
				"%% date: ",
				"%% tags: test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(&tt.metadata)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestUpdate_ReplaceExisting(t *testing.T) {
	content := `%% Metadata
%% title: Old Title
%% date: 2024-01-01
%% tags: old

\documentclass{article}
\begin{document}
Content here
\end{document}`

	newMetadata := &Metadata{
		Title: "New Title",
		Date:  "2024-01-15",
		Tags:  []string{"new", "updated"},
	}

	result := Update(content, newMetadata)

	// Should contain new metadata
	if !strings.Contains(result, "%% title: New Title") {
		t.Error("Expected new title in result")
	}
	if !strings.Contains(result, "%% date: 2024-01-15") {
		t.Error("Expected new date in result")
	}
	if !strings.Contains(result, "%% tags: new, updated") {
		t.Error("Expected new tags in result")
	}

	// Should not contain old metadata
	if strings.Contains(result, "Old Title") {
		t.Error("Should not contain old title")
	}

	// Should preserve content
	if !strings.Contains(result, "\\documentclass{article}") {
		t.Error("Should preserve document content")
	}
	if !strings.Contains(result, "Content here") {
		t.Error("Should preserve document content")
	}
}

func TestUpdate_PrependWhenMissing(t *testing.T) {
	content := `\documentclass{article}
\begin{document}
Content without metadata
\end{document}`

	newMetadata := &Metadata{
		Title: "Added Title",
		Date:  "2024-01-15",
		Tags:  []string{"new"},
	}

	result := Update(content, newMetadata)

	// Should start with metadata
	lines := strings.Split(result, "\n")
	if !strings.Contains(lines[0], "%% Metadata") {
		t.Error("Expected metadata to be prepended")
	}

	// Should contain new metadata
	if !strings.Contains(result, "%% title: Added Title") {
		t.Error("Expected new title in result")
	}

	// Should preserve original content
	if !strings.Contains(result, "\\documentclass{article}") {
		t.Error("Should preserve original content")
	}
}

func TestExtract_Convenience(t *testing.T) {
	content := `%% Metadata
%% title: Test Note
%% date: 2024-01-15
%% tags: test

\documentclass{article}`

	metadata, err := Extract(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got %q", metadata.Title)
	}
	if metadata.Date != "2024-01-15" {
		t.Errorf("Expected date '2024-01-15', got %q", metadata.Date)
	}
}

func TestExtractStrict_Convenience(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid metadata",
			content: `%% Metadata
%% title: Test Note
%% date: 2024-01-15
%% tags: test

\documentclass{article}`,
			wantErr: false,
		},
		{
			name: "missing title",
			content: `%% Metadata
%% date: 2024-01-15
%% tags: test

\documentclass{article}`,
			wantErr: true,
		},
		{
			name: "invalid date",
			content: `%% Metadata
%% title: Test Note
%% date: invalid-date
%% tags: test

\documentclass{article}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractStrict(tt.content)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParser_Parse_EmptyTitle(t *testing.T) {
	content := `%% Metadata
%% title:
%% date: 2024-01-15
%% tags: test

\documentclass{article}`

	parser := NewParser(true)
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for empty title in strict mode")
	}
}

func TestParser_ValidateDate(t *testing.T) {
	parser := NewParser(false)

	tests := []struct {
		date    string
		wantErr bool
	}{
		{"2024-01-15", false},
		{"2024-12-31", false},
		{"2024-02-29", false}, // Leap year
		{"2023-02-29", true},  // Not a leap year
		{"2024-13-01", true},  // Invalid month
		{"2024-01-32", true},  // Invalid day
		{"01/15/2024", true},  // Wrong format
		{"2024/01/15", true},  // Wrong format
		{"not-a-date", true},
		{"", false}, // Empty is allowed
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			err := parser.validateDate(tt.date)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for date %q", tt.date)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for date %q: %v", tt.date, err)
			}
		})
	}
}

func TestFormat_AutoDate(t *testing.T) {
	metadata := &Metadata{
		Title: "Test",
		Date:  "",
		Tags:  []string{},
	}

	result := Format(metadata)

	// Should contain today's date in YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(result, "%% date: "+today) {
		t.Errorf("Expected today's date (%s) to be added, got:\n%s", today, result)
	}
}

func TestParser_Normalization(t *testing.T) {
	content := `%% Metadata
%% title:   Title With Spaces
%% date:   2024-01-15
%% tags:   tag1  ,  TAG1  , tag2 , Tag2

\documentclass{article}`

	parser := NewParser(false)
	result, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Title should be trimmed
	if result.Metadata.Title != "Title With Spaces" {
		t.Errorf("Expected trimmed title, got %q", result.Metadata.Title)
	}

	// Date should be trimmed
	if result.Metadata.Date != "2024-01-15" {
		t.Errorf("Expected trimmed date, got %q", result.Metadata.Date)
	}

	// Duplicate tags (case-insensitive) should be removed
	if len(result.Metadata.Tags) != 2 {
		t.Errorf("Expected 2 unique tags, got %d: %v", len(result.Metadata.Tags), result.Metadata.Tags)
	}
}

func TestParseError_Error(t *testing.T) {
	err := ParseError{
		Line:    5,
		Field:   "title",
		Message: "title cannot be empty",
	}

	expected := "line 5 (title): title cannot be empty"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}
