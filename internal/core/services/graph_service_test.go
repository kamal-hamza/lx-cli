package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
)

func TestGraphService_GetGraph_Empty(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.json")

	// Create real IndexerService (it's logic-heavy but depends on Repo, which we mock)
	indexer := NewIndexerService(mockRepo, indexPath)
	svc := NewGraphService(indexer)

	// Execute
	graph, err := svc.GetGraph(context.Background(), true)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(graph.Links))
	}
}

func TestGraphService_GetGraph_SingleNote(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.json")

	indexer := NewIndexerService(mockRepo, indexPath)
	svc := NewGraphService(indexer)

	// Create 1 Note
	header, _ := domain.NewNoteHeader("Solo Note", []string{"tag1"})
	note := domain.NewNoteBody(header, "Content with no links")
	mockRepo.Save(context.Background(), note)

	// Execute
	graph, err := svc.GetGraph(context.Background(), true)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}

	if graph.Nodes[0].ID != header.Slug {
		t.Errorf("expected node ID %s, got %s", header.Slug, graph.Nodes[0].ID)
	}

	if len(graph.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(graph.Links))
	}
}

func TestGraphService_GetGraph_WithLinks(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.json")

	indexer := NewIndexerService(mockRepo, indexPath)
	svc := NewGraphService(indexer)

	// Note A: References B
	headerA, _ := domain.NewNoteHeader("Source Note", []string{"tagA"})
	// We use the slug 'target-note' which corresponds to "Target Note"
	noteA := domain.NewNoteBody(headerA, `See \ref{target-note} for details`)
	mockRepo.Save(context.Background(), noteA)

	// Note B: Target
	headerB, _ := domain.NewNoteHeader("Target Note", []string{"tagB"})
	noteB := domain.NewNoteBody(headerB, "Content")
	mockRepo.Save(context.Background(), noteB)

	// Execute
	graph, err := svc.GetGraph(context.Background(), true)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.Nodes))
	}

	if len(graph.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(graph.Links))
	}

	link := graph.Links[0]
	if link.Source != headerA.Slug {
		t.Errorf("expected link source %s, got %s", headerA.Slug, link.Source)
	}
	if link.Target != headerB.Slug {
		t.Errorf("expected link target %s, got %s", headerB.Slug, link.Target)
	}
}

func TestGraphService_GetGraph_BrokenLink(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.json")

	indexer := NewIndexerService(mockRepo, indexPath)
	svc := NewGraphService(indexer)

	// Note references a non-existent note
	header, _ := domain.NewNoteHeader("Broken Link Note", []string{})
	note := domain.NewNoteBody(header, `See \ref{does-not-exist}`)
	mockRepo.Save(context.Background(), note)

	// Execute
	graph, err := svc.GetGraph(context.Background(), true)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 node (the source)
	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Nodes))
	}

	// Should have 0 links (graph should not render links to missing nodes)
	if len(graph.Links) != 0 {
		t.Errorf("expected 0 links (broken link ignored), got %d", len(graph.Links))
	}
}
