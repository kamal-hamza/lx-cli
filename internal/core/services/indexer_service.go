package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"lx/internal/core/domain"
	"lx/internal/core/ports"
)

// IndexerService handles indexing operations for the vault
type IndexerService struct {
	noteRepo  ports.Repository
	indexPath string
}

// NewIndexerService creates a new indexer service
func NewIndexerService(noteRepo ports.Repository, indexPath string) *IndexerService {
	return &IndexerService{
		noteRepo:  noteRepo,
		indexPath: indexPath,
	}
}

// ReindexRequest represents a request to reindex the vault
type ReindexRequest struct {
	// Reserved for future options (e.g., incremental indexing)
}

// ReindexResponse represents the response from reindexing
type ReindexResponse struct {
	TotalNotes       int
	TotalConnections int
	Duration         string
}

// linkPattern matches LaTeX link commands
// Captures: \input{file}, \include{file}, \ref{label}, \cref{label}, \cite{key}
var linkPattern = regexp.MustCompile(`\\(?:input|include|ref|cref|cite)\{([^}]+)\}`)

// Execute performs a full reindex of the vault
func (s *IndexerService) Execute(ctx context.Context, req ReindexRequest) (*ReindexResponse, error) {
	// Create new index
	index := domain.NewIndex()

	// Pass 1: Extract metadata and outgoing links
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	// Build index entries
	for _, header := range headers {
		// Get full note content to parse links
		note, err := s.noteRepo.Get(ctx, header.Slug)
		if err != nil {
			// Skip notes we can't read, but don't fail the entire operation
			continue
		}

		// Extract outgoing links from content
		outgoingLinks := s.extractLinks(note.Content, header.Slug)

		// Create index entry
		entry := domain.IndexEntry{
			Title:         header.Title,
			Date:          header.Date,
			Tags:          header.Tags,
			Filename:      header.Filename,
			OutgoingLinks: outgoingLinks,
			Backlinks:     []string{}, // Will be populated in Pass 2
		}

		index.AddNote(header.Slug, entry)
	}

	// Pass 2: Calculate backlinks by inverting the outgoing links
	s.calculateBacklinks(index)

	// Update timestamp
	index.UpdateLastIndexed()

	// Save to disk
	if err := s.saveIndex(index); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	return &ReindexResponse{
		TotalNotes:       index.Count(),
		TotalConnections: index.CountConnections(),
	}, nil
}

// extractLinks extracts all LaTeX link references from content
func (s *IndexerService) extractLinks(content string, sourceSlug string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)

	// Use map to deduplicate
	linkMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			target := match[1]

			// Normalize the link target to extract slug
			slug := s.normalizeLink(target)

			// Don't add self-references
			if slug != "" && slug != sourceSlug {
				linkMap[slug] = true
			}
		}
	}

	// Convert map to slice
	var links []string
	for link := range linkMap {
		links = append(links, link)
	}

	return links
}

// normalizeLink converts a LaTeX link reference to a slug
// Examples:
//   - "../notes/graph-theory.tex" -> "graph-theory"
//   - "graph-theory.tex" -> "graph-theory"
//   - "graph-theory" -> "graph-theory"
func (s *IndexerService) normalizeLink(link string) string {
	// Convert backslashes to forward slashes for Windows compatibility
	link = strings.ReplaceAll(link, "\\", "/")

	// Remove path components
	link = filepath.Base(link)

	// Remove .tex extension
	link = strings.TrimSuffix(link, ".tex")

	// Remove date prefix if present (YYYYMMDD-)
	if len(link) > 9 && link[8] == '-' {
		// Check if first 8 chars are digits
		isDate := true
		for i := 0; i < 8; i++ {
			if link[i] < '0' || link[i] > '9' {
				isDate = false
				break
			}
		}
		if isDate {
			link = link[9:] // Skip "YYYYMMDD-"
		}
	}

	return strings.TrimSpace(link)
}

// calculateBacklinks populates backlinks by inverting outgoing links
func (s *IndexerService) calculateBacklinks(index *domain.Index) {
	// First, clear all backlinks
	for slug, entry := range index.Notes {
		entry.Backlinks = []string{}
		index.AddNote(slug, entry)
	}

	// For each note, add itself to the backlinks of all its targets
	for sourceSlug, entry := range index.Notes {
		for _, targetSlug := range entry.OutgoingLinks {
			if target, exists := index.GetNote(targetSlug); exists {
				// Add sourceSlug to target's backlinks
				target.Backlinks = append(target.Backlinks, sourceSlug)
				index.AddNote(targetSlug, target)
			}
		}
	}
}

// saveIndex writes the index to disk
func (s *IndexerService) saveIndex(index *domain.Index) error {
	// Ensure directory exists
	dir := filepath.Dir(s.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// LoadIndex loads the index from disk
func (s *IndexerService) LoadIndex() (*domain.Index, error) {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty index if file doesn't exist
			return domain.NewIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index domain.Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

// IndexExists checks if the index file exists
func (s *IndexerService) IndexExists() bool {
	_, err := os.Stat(s.indexPath)
	return err == nil
}
