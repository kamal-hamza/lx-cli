package domain

import (
	"time"
)

// Index represents the persistent cache of vault metadata and connections
type Index struct {
	Version     string                `json:"version"`
	LastIndexed time.Time             `json:"last_indexed"`
	Notes       map[string]IndexEntry `json:"notes"`
}

// IndexEntry represents cached metadata and connections for a single note
type IndexEntry struct {
	Title    string   `json:"title"`
	Date     string   `json:"date"`
	Tags     []string `json:"tags"`
	Filename string   `json:"filename"`

	// Graph Connections
	OutgoingLinks []string `json:"outgoing_links"` // \ref, \input, etc.
	Backlinks     []string `json:"backlinks"`      // Other files pointing here

	// Assets used in this note
	Assets []string `json:"assets"` // \includegraphics{...}
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{
		Version:     "1.1",
		LastIndexed: time.Now(),
		Notes:       make(map[string]IndexEntry),
	}
}

// AddNote adds or updates a note in the index
func (idx *Index) AddNote(slug string, entry IndexEntry) {
	if idx.Notes == nil {
		idx.Notes = make(map[string]IndexEntry)
	}
	idx.Notes[slug] = entry
}

// GetNote retrieves a note from the index
func (idx *Index) GetNote(slug string) (IndexEntry, bool) {
	entry, exists := idx.Notes[slug]
	return entry, exists
}

// HasNote checks if a note exists in the index
func (idx *Index) HasNote(slug string) bool {
	_, exists := idx.Notes[slug]
	return exists
}

// Count returns the total number of notes in the index
func (idx *Index) Count() int {
	return len(idx.Notes)
}

// CountConnections returns the total number of connections (links) in the graph
func (idx *Index) CountConnections() int {
	count := 0
	for _, entry := range idx.Notes {
		count += len(entry.OutgoingLinks)
	}
	return count
}

// UpdateLastIndexed updates the last indexed timestamp
func (idx *Index) UpdateLastIndexed() {
	idx.LastIndexed = time.Now()
}

// Clear removes all notes from the index
func (idx *Index) Clear() {
	idx.Notes = make(map[string]IndexEntry)
}
