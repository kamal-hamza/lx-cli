package services

import (
	"testing"
)

func TestExtractLinks(t *testing.T) {
	// Create a mock indexer for testing
	indexer := &IndexerService{}

	tests := []struct {
		name       string
		content    string
		sourceSlug string
		expected   []string
	}{
		{
			name: "input command",
			content: `\documentclass{article}
\begin{document}
\input{../notes/graph-theory.tex}
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"graph-theory"},
		},
		{
			name: "multiple ref commands",
			content: `\documentclass{article}
\begin{document}
See \ref{linear-algebra} and \ref{set-theory} for details.
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"linear-algebra", "set-theory"},
		},
		{
			name: "cite command",
			content: `\documentclass{article}
\begin{document}
According to \cite{neural-networks}, deep learning works.
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"neural-networks"},
		},
		{
			name: "mixed commands",
			content: `\documentclass{article}
\begin{document}
\input{topology.tex}
See \ref{algebra} and \cite{logic}.
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"topology", "algebra", "logic"},
		},
		{
			name: "self-reference ignored",
			content: `\documentclass{article}
\begin{document}
\ref{graph-theory}
\end{document}`,
			sourceSlug: "graph-theory",
			expected:   []string{},
		},
		{
			name: "duplicate links deduplicated",
			content: `\documentclass{article}
\begin{document}
\ref{algebra}
\ref{algebra}
\cite{algebra}
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"algebra"},
		},
		{
			name: "include command",
			content: `\documentclass{article}
\begin{document}
\include{chapter1}
\end{document}`,
			sourceSlug: "main",
			expected:   []string{"chapter1"},
		},
		{
			name: "cref command",
			content: `\documentclass{article}
\begin{document}
As shown in \cref{theorem-1}, we have proof.
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{"theorem-1"},
		},
		{
			name: "no links",
			content: `\documentclass{article}
\begin{document}
Just plain content.
\end{document}`,
			sourceSlug: "current-note",
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexer.extractLinks(tt.content, tt.sourceSlug)

			// Convert result to map for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, link := range result {
				resultMap[link] = true
			}

			expectedMap := make(map[string]bool)
			for _, link := range tt.expected {
				expectedMap[link] = true
			}

			// Check length
			if len(resultMap) != len(expectedMap) {
				t.Errorf("extractLinks() returned %d links, expected %d\nGot: %v\nExpected: %v",
					len(resultMap), len(expectedMap), result, tt.expected)
				return
			}

			// Check each expected link exists
			for link := range expectedMap {
				if !resultMap[link] {
					t.Errorf("extractLinks() missing expected link: %s\nGot: %v\nExpected: %v",
						link, result, tt.expected)
				}
			}
		})
	}
}

func TestNormalizeLink(t *testing.T) {
	indexer := &IndexerService{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "relative path with .tex",
			input:    "../notes/graph-theory.tex",
			expected: "graph-theory",
		},
		{
			name:     "filename with .tex",
			input:    "graph-theory.tex",
			expected: "graph-theory",
		},
		{
			name:     "plain slug",
			input:    "graph-theory",
			expected: "graph-theory",
		},
		{
			name:     "dated filename",
			input:    "20251128-graph-theory.tex",
			expected: "graph-theory",
		},
		{
			name:     "dated filename without extension",
			input:    "20251128-graph-theory",
			expected: "graph-theory",
		},
		{
			name:     "absolute path",
			input:    "/home/user/notes/algebra.tex",
			expected: "algebra",
		},
		{
			name:     "windows path",
			input:    "C:\\notes\\topology.tex",
			expected: "topology",
		},
		{
			name:     "slug with hyphens",
			input:    "linear-algebra-basics",
			expected: "linear-algebra-basics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexer.normalizeLink(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLink(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
