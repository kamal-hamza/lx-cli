package metadata

import (
	"reflect"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    *Metadata
		expected string
	}{
		{
			name: "complete metadata",
			input: &Metadata{
				Title: "Graph Theory",
				Date:  "2025-12-14",
				Tags:  []string{"math", "graphs"},
			},
			expected: "% ---\n% title: Graph Theory\n% date: 2025-12-14\n% tags: math, graphs\n% ---\n",
		},
		{
			name: "no tags",
			input: &Metadata{
				Title: "Simple Note",
				Date:  "2025-01-01",
				Tags:  []string{},
			},
			expected: "% ---\n% title: Simple Note\n% date: 2025-01-01\n% ---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.input)
			if got != tt.expected {
				t.Errorf("Format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Metadata
		wantErr bool
	}{
		{
			name: "standard header",
			content: `% ---
% title: Graph Theory
% date: 2025-12-14
% tags: math, school
% ---
\documentclass{article}`,
			want: &Metadata{
				Title: "Graph Theory",
				Date:  "2025-12-14",
				Tags:  []string{"math", "school"},
			},
			wantErr: false,
		},
		{
			name: "extra whitespace",
			content: `% ---
%   title:    Spaces Everywhere
%date:2025-01-01
% tags:  tag1  ,  tag2
% ---`,
			want: &Metadata{
				Title: "Spaces Everywhere",
				Date:  "2025-01-01",
				Tags:  []string{"tag1", "tag2"},
			},
			wantErr: false,
		},
		{
			name: "missing tags (valid)",
			content: `% ---
% title: No Tags Here
% date: 2025-01-01
% ---`,
			want: &Metadata{
				Title: "No Tags Here",
				Date:  "2025-01-01",
				Tags:  []string{},
			},
			wantErr: false,
		},
		{
			name: "missing title (error)",
			content: `% ---
% date: 2025-01-01
% ---`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Extract(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Extract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractStrict(t *testing.T) {
	// Strict parsing requires Date field
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "has date",
			content: `% title: OK
% date: 2025-01-01`,
			wantErr: false,
		},
		{
			name: "missing date",
			content: `% title: Fail
% tags: none`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractStrict(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		newTitle string
		want     string
		wantErr  bool
	}{
		{
			name: "standard update",
			content: `% ---
% title: Old Title
% date: 2025-01-01
% ---`,
			newTitle: "New Title",
			want: `% ---
% title: New Title
% date: 2025-01-01
% ---`,
			wantErr: false,
		},
		{
			name: "preserves spacing",
			content: `% ---
%   title:    Old Title
% date: 2025-01-01
% ---`,
			newTitle: "New Title",
			want: `% ---
%   title:    New Title
% date: 2025-01-01
% ---`,
			wantErr: false,
		},
		{
			name: "no title found",
			content: `% ---
% date: 2025-01-01
% ---`,
			newTitle: "New Title",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateTitle(tt.content, tt.newTitle)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTitle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("UpdateTitle() = \n%q\nwant \n%q", got, tt.want)
			}
		})
	}
}
