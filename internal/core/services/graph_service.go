package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/ports"
	"github.com/kamal-hamza/lx-cli/pkg/config"
)

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
	ID    string
	Title string
	Tags  []string
}

// GraphLink represents a connection between two nodes
type GraphLink struct {
	Source string
	Target string
}

// GraphData contains the full graph structure
type GraphData struct {
	Nodes []GraphNode
	Links []GraphLink
}

type GraphService struct {
	repo   ports.Repository
	config *config.Config
}

// NewGraphService creates a new instance of GraphService with config
func NewGraphService(repo ports.Repository, cfg *config.Config) *GraphService {
	return &GraphService{
		repo:   repo,
		config: cfg,
	}
}

// GenerateGraphDOT generates a DOT format string for the note graph
func (s *GraphService) GenerateGraphDOT(ctx context.Context) (string, error) {
	notes, err := s.repo.ListHeaders(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list notes for graph: %w", err)
	}

	// 1. Apply MaxNodes Limit
	maxNodes := s.config.GraphMaxNodes
	if maxNodes > 0 && len(notes) > maxNodes {
		notes = notes[:maxNodes]
	}

	// 2. Build map for quick lookup
	existingSlugs := make(map[string]bool)
	for _, n := range notes {
		existingSlugs[n.Slug] = true
	}

	var sb strings.Builder

	// 3. Apply Direction Config
	direction := s.config.GraphDirection
	if direction == "" {
		direction = "LR"
	}

	sb.WriteString("digraph G {\n")
	sb.WriteString(fmt.Sprintf("  rankdir=%s;\n", direction))
	sb.WriteString("  node [shape=box, style=filled, fillcolor=\"#f9f9f9\", fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [color=\"#555555\"];\n")

	for _, note := range notes {
		safeSlug := strings.ReplaceAll(note.Slug, "-", "_")
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", safeSlug, note.Title))

		for _, link := range note.Tags {
			if strings.HasPrefix(link, "link:") {
				target := strings.TrimPrefix(link, "link:")
				if existingSlugs[target] {
					safeTarget := strings.ReplaceAll(target, "-", "_")
					sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", safeSlug, safeTarget))
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String(), nil
}

// GetGraph returns the graph data structure for interactive visualization
func (s *GraphService) GetGraph(ctx context.Context, forceRefresh bool) (GraphData, error) {
	notes, err := s.repo.ListHeaders(ctx)
	if err != nil {
		return GraphData{}, fmt.Errorf("failed to list notes for graph: %w", err)
	}

	// Apply MaxNodes Limit
	maxNodes := s.config.GraphMaxNodes
	if maxNodes > 0 && len(notes) > maxNodes {
		notes = notes[:maxNodes]
	}

	// Build graph data
	var nodes []GraphNode
	var links []GraphLink
	existingSlugs := make(map[string]bool)

	// First pass: create nodes
	for _, note := range notes {
		nodes = append(nodes, GraphNode{
			ID:    note.Slug,
			Title: note.Title,
			Tags:  note.Tags,
		})
		existingSlugs[note.Slug] = true
	}

	// Second pass: create links
	// Look for various link patterns in the content
	for _, note := range notes {
		// Get the full note to analyze links
		fullNote, err := s.repo.Get(ctx, note.Slug)
		if err != nil {
			continue // Skip if we can't read the note
		}

		// Extract links from content
		linkedSlugs := extractLinksFromContent(fullNote.Content)
		for _, targetSlug := range linkedSlugs {
			// Only create link if target exists
			if existingSlugs[targetSlug] {
				links = append(links, GraphLink{
					Source: note.Slug,
					Target: targetSlug,
				})
			}
		}

		// Also check tags for link: prefix
		for _, tag := range note.Tags {
			if strings.HasPrefix(tag, "link:") {
				targetSlug := strings.TrimPrefix(tag, "link:")
				if existingSlugs[targetSlug] {
					links = append(links, GraphLink{
						Source: note.Slug,
						Target: targetSlug,
					})
				}
			}
		}
	}

	return GraphData{
		Nodes: nodes,
		Links: links,
	}, nil
}

// extractLinksFromContent extracts note references from LaTeX content
func extractLinksFromContent(content string) []string {
	var links []string
	seen := make(map[string]bool)

	// Look for common LaTeX reference patterns
	patterns := []string{
		`\ref{`,
		`\cite{`,
		`\input{`,
		`\include{`,
	}

	for _, pattern := range patterns {
		idx := 0
		for {
			idx = strings.Index(content[idx:], pattern)
			if idx == -1 {
				break
			}
			idx += len(pattern)

			// Find the closing brace
			end := strings.Index(content[idx:], "}")
			if end == -1 {
				break
			}

			ref := content[idx : idx+end]
			// Clean up the reference to get just the slug
			ref = strings.TrimSpace(ref)
			ref = strings.TrimSuffix(ref, ".tex")

			// Extract just the filename if it's a path
			if strings.Contains(ref, "/") {
				parts := strings.Split(ref, "/")
				ref = parts[len(parts)-1]
			}

			if ref != "" && !seen[ref] {
				links = append(links, ref)
				seen[ref] = true
			}

			idx += end + 1
		}
	}

	return links
}
