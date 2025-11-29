package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
)

type GraphService struct {
	indexer *IndexerService
}

func NewGraphService(indexer *IndexerService) *GraphService {
	return &GraphService{
		indexer: indexer,
	}
}

// Data Structures (Keep existing JSON contract)
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Group int    `json:"group"`
	Value int    `json:"val"`
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

// GetGraph retrieves the graph data from the Index
func (s *GraphService) GetGraph(ctx context.Context, forceRefresh bool) (GraphData, error) {
	// 1. Ensure Index is loaded/fresh
	var index *domain.Index
	var err error

	if forceRefresh || !s.indexer.IndexExists() {
		// Rebuild if forced or missing
		_, err := s.indexer.Execute(ctx, ReindexRequest{})
		if err != nil {
			return GraphData{}, fmt.Errorf("failed to generate index: %w", err)
		}
	}

	index, err = s.indexer.LoadIndex()
	if err != nil {
		return GraphData{}, err
	}

	// 2. Transform Index to GraphData
	nodes := make([]GraphNode, 0, len(index.Notes))
	links := make([]GraphLink, 0)

	// Track connections for node sizing
	connectionCounts := make(map[string]int)

	// Create Links
	for sourceSlug, entry := range index.Notes {
		// Outgoing links
		for _, targetSlug := range entry.OutgoingLinks {
			// Ensure target exists in index (avoid broken links in graph)
			if _, exists := index.Notes[targetSlug]; exists {
				links = append(links, GraphLink{
					Source: sourceSlug,
					Target: targetSlug,
					Value:  1,
				})
				connectionCounts[sourceSlug]++
				connectionCounts[targetSlug]++
			}
		}
	}

	// Create Nodes
	for slug, entry := range index.Notes {
		val := connectionCounts[slug]
		if val == 0 {
			val = 1 // Minimum size
		}

		// Simple grouping by first tag, or default 1
		group := 1
		if len(entry.Tags) > 0 {
			// Simple hash of tag string to int group
			group = int(entry.Tags[0][0])
		}

		nodes = append(nodes, GraphNode{
			ID:    slug,
			Title: entry.Title,
			Group: group,
			Value: val,
		})
	}

	// Sort for deterministic output
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	return GraphData{
		Nodes: nodes,
		Links: links,
	}, nil
}
