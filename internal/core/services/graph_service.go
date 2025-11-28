package services

import (
	"context"
	"fmt"
	"strings"

	"lx/internal/core/domain"
)

// GraphService handles graph visualization operations
type GraphService struct {
	indexer *IndexerService
}

// NewGraphService creates a new graph service
func NewGraphService(indexer *IndexerService) *GraphService {
	return &GraphService{
		indexer: indexer,
	}
}

// GenerateRequest represents a request to generate a graph
type GenerateRequest struct {
	Format string // "dot" (default), future: "json", "mermaid"
}

// GenerateResponse represents the response from graph generation
type GenerateResponse struct {
	Format string
	Output string
}

// Execute generates a graph visualization
func (s *GraphService) Execute(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// Load or reindex if needed
	index, err := s.ensureIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Default to DOT format
	format := req.Format
	if format == "" {
		format = "dot"
	}

	var output string
	switch format {
	case "dot":
		output = s.generateDOT(index)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return &GenerateResponse{
		Format: format,
		Output: output,
	}, nil
}

// ensureIndex loads the index or triggers a reindex if it doesn't exist
func (s *GraphService) ensureIndex(ctx context.Context) (*domain.Index, error) {
	if !s.indexer.IndexExists() {
		// Trigger reindex
		if _, err := s.indexer.Execute(ctx, ReindexRequest{}); err != nil {
			return nil, fmt.Errorf("failed to reindex: %w", err)
		}
	}

	return s.indexer.LoadIndex()
}

// generateDOT generates a Graphviz DOT format representation
func (s *GraphService) generateDOT(index *domain.Index) string {
	var builder strings.Builder

	// Header
	builder.WriteString("digraph G {\n")
	builder.WriteString("  rankdir=LR;\n")
	builder.WriteString("  node [shape=box, style=rounded];\n")
	builder.WriteString("\n")

	// Add nodes with labels
	for slug, entry := range index.Notes {
		// Escape quotes in title
		title := strings.ReplaceAll(entry.Title, "\"", "\\\"")
		builder.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", slug, title))
	}

	builder.WriteString("\n")

	// Add edges (connections)
	for sourceSlug, entry := range index.Notes {
		for _, targetSlug := range entry.OutgoingLinks {
			// Only add edge if target exists in the index
			if index.HasNote(targetSlug) {
				builder.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", sourceSlug, targetSlug))
			}
		}
	}

	// Footer
	builder.WriteString("}\n")

	return builder.String()
}

// GetBacklinks returns all notes that link to the specified slug
func (s *GraphService) GetBacklinks(ctx context.Context, slug string) ([]string, error) {
	index, err := s.ensureIndex(ctx)
	if err != nil {
		return nil, err
	}

	entry, exists := index.GetNote(slug)
	if !exists {
		return []string{}, nil
	}

	return entry.Backlinks, nil
}

// GetOutgoingLinks returns all notes that the specified slug links to
func (s *GraphService) GetOutgoingLinks(ctx context.Context, slug string) ([]string, error) {
	index, err := s.ensureIndex(ctx)
	if err != nil {
		return nil, err
	}

	entry, exists := index.GetNote(slug)
	if !exists {
		return []string{}, nil
	}

	return entry.OutgoingLinks, nil
}
