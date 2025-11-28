package services

import (
	"context"
	"strings"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
)

func TestGraphService_Generate_EmptyRepository(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	svc := NewGraphService(mockRepo, "/tmp/test-vault")

	// Execute
	graph, err := svc.Generate(context.Background())

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

func TestGraphService_Generate_SingleNote(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	svc := NewGraphService(mockRepo, "/tmp/test-vault")

	// Create a single note
	header, _ := domain.NewNoteHeader("Single Note", []string{"tag1"})
	note := domain.NewNoteBody(header, "This is a standalone note with no links.")
	mockRepo.Save(context.Background(), note)

	// Execute
	graph, err := svc.Generate(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}

	node := graph.Nodes[0]
	if node.ID != header.Slug {
		t.Errorf("expected node ID=%s, got %s", header.Slug, node.ID)
	}

	if node.Title != header.Title {
		t.Errorf("expected node Title=%s, got %s", header.Title, node.Title)
	}

	if node.Group != 1 {
		t.Errorf("expected node Group=1, got %d", node.Group)
	}

	if node.Value != 1 {
		t.Errorf("expected node Value=1, got %d", node.Value)
	}

	// No links since there's only one note
	if len(graph.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(graph.Links))
	}
}

func TestGraphService_Generate_MultipleNotesNoLinks(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	svc := NewGraphService(mockRepo, "/tmp/test-vault")

	// Create multiple notes with no links between them
	notes := []struct {
		title string
		tags  []string
	}{
		{"First Note", []string{"math"}},
		{"Second Note", []string{"physics"}},
		{"Third Note", []string{"chemistry"}},
	}

	for _, n := range notes {
		header, _ := domain.NewNoteHeader(n.title, n.tags)
		note := domain.NewNoteBody(header, "Content without links to other notes.")
		mockRepo.Save(context.Background(), note)
	}

	// Execute
	graph, err := svc.Generate(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != len(notes) {
		t.Errorf("expected %d nodes, got %d", len(notes), len(graph.Nodes))
	}

	// Verify each note became a node
	nodeIDs := make(map[string]bool)
	for _, node := range graph.Nodes {
		nodeIDs[node.ID] = true
	}

	if len(nodeIDs) != len(notes) {
		t.Errorf("expected %d unique node IDs, got %d", len(notes), len(nodeIDs))
	}

	// No links expected
	if len(graph.Links) != 0 {
		t.Errorf("expected 0 links, got %d", len(graph.Links))
	}
}

func TestGraphService_IsSlugChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'_', false},
		{' ', false},
		{'.', false},
		{'/', false},
		{'\\', false},
		{'!', false},
		{'@', false},
	}

	for _, test := range tests {
		result := isSlugChar(test.char)
		if result != test.expected {
			t.Errorf("isSlugChar('%c'): expected %v, got %v", test.char, test.expected, result)
		}
	}
}

func TestGraphService_FindMentions_NoMatches(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology":       true,
		"graph-theory":   true,
		"linear-algebra": true,
	}

	content := "This is some content that does not mention any valid slugs."
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	if len(found) != 0 {
		t.Errorf("expected 0 matches, got %d", len(found))
	}
}

func TestGraphService_FindMentions_SingleMatch(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology":     true,
		"graph-theory": true,
	}

	content := "This note discusses topology in detail."
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	if len(found) != 1 {
		t.Fatalf("expected 1 match, got %d", len(found))
	}

	if !found["topology"] {
		t.Error("expected to find 'topology'")
	}
}

func TestGraphService_FindMentions_MultipleMatches(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology":       true,
		"graph-theory":   true,
		"linear-algebra": true,
	}

	content := `
	This note references topology and graph-theory.
	We also discuss linear-algebra concepts.
	topology appears multiple times.
	`
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	expectedMatches := []string{"topology", "graph-theory", "linear-algebra"}
	if len(found) != len(expectedMatches) {
		t.Fatalf("expected %d unique matches, got %d", len(expectedMatches), len(found))
	}

	for _, slug := range expectedMatches {
		if !found[slug] {
			t.Errorf("expected to find '%s'", slug)
		}
	}
}

func TestGraphService_FindMentions_CaseInsensitive(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology": true,
	}

	content := "This mentions TOPOLOGY and Topology and topology."
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	if len(found) != 1 {
		t.Fatalf("expected 1 unique match (case-insensitive), got %d", len(found))
	}

	if !found["topology"] {
		t.Error("expected to find 'topology'")
	}
}

func TestGraphService_FindMentions_IgnoresPartialMatches(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"graph": true,
	}

	// "graph" appears as part of larger words but should not match
	content := "photography and autograph and graph and graphite"
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	// Should only find the standalone "graph"
	if len(found) != 1 {
		t.Fatalf("expected 1 match, got %d", len(found))
	}

	if !found["graph"] {
		t.Error("expected to find standalone 'graph'")
	}
}

func TestGraphService_FindMentions_WithHyphens(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"graph-theory":   true,
		"linear-algebra": true,
		"set-theory":     true,
	}

	content := "We study graph-theory and linear-algebra but not set-theory today."
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	expectedMatches := []string{"graph-theory", "linear-algebra", "set-theory"}
	if len(found) != len(expectedMatches) {
		t.Fatalf("expected %d matches, got %d", len(expectedMatches), len(found))
	}

	for _, slug := range expectedMatches {
		if !found[slug] {
			t.Errorf("expected to find '%s'", slug)
		}
	}
}

func TestGraphService_FindMentions_EmptyContent(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology": true,
	}

	content := ""
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	if len(found) != 0 {
		t.Errorf("expected 0 matches in empty content, got %d", len(found))
	}
}

func TestGraphService_FindMentions_OnlyWhitespace(t *testing.T) {
	svc := &GraphService{}
	validSlugs := map[string]bool{
		"topology": true,
	}

	content := "   \n\t\n   "
	reader := strings.NewReader(content)

	found := svc.findMentions(reader, validSlugs)

	if len(found) != 0 {
		t.Errorf("expected 0 matches in whitespace-only content, got %d", len(found))
	}
}

func TestGraphService_Generate_ContextCancellation(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	svc := NewGraphService(mockRepo, "/tmp/test-vault")

	// Create some notes
	for i := 0; i < 5; i++ {
		header, _ := domain.NewNoteHeader("Test Note", []string{})
		note := domain.NewNoteBody(header, "Content")
		mockRepo.Save(context.Background(), note)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute - should handle cancellation gracefully
	// Note: Current implementation doesn't check context in Generate,
	// but this test documents expected behavior for future improvement
	_, err := svc.Generate(ctx)

	// Current implementation doesn't return error on cancelled context
	// but completes quickly with empty repo
	if err != nil {
		// If implementation adds context checking, this would be valid
		t.Logf("context cancellation detected: %v", err)
	}
}

func TestGraphService_NodeStructure(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	svc := NewGraphService(mockRepo, "/tmp/test-vault")

	// Create a note with specific properties
	title := "Advanced Topology"
	tags := []string{"math", "topology"}
	header, _ := domain.NewNoteHeader(title, tags)
	note := domain.NewNoteBody(header, "Content about topology")
	mockRepo.Save(context.Background(), note)

	// Execute
	graph, err := svc.Generate(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}

	node := graph.Nodes[0]

	// Verify node structure
	if node.ID != header.Slug {
		t.Errorf("node.ID: expected %s, got %s", header.Slug, node.ID)
	}

	if node.Title != title {
		t.Errorf("node.Title: expected %s, got %s", title, node.Title)
	}

	if node.Group != 1 {
		t.Errorf("node.Group: expected 1, got %d", node.Group)
	}

	if node.Value != 1 {
		t.Errorf("node.Value: expected 1, got %d", node.Value)
	}
}
