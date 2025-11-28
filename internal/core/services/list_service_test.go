package services

import (
	"context"
	"strings"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
)

func TestListService_Execute(t *testing.T) {
	tests := []struct {
		name          string
		request       ListRequest
		setupMocks    func(*mocks.MockRepository)
		expectedCount int
		expectError   bool
	}{
		{
			name: "list all notes",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{"math"})
				createTestNote(repo, "Note 2", []string{"science"})
				createTestNote(repo, "Note 3", []string{"history"})
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "empty vault",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks:    func(repo *mocks.MockRepository) {},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "filter by tag - single match",
			request: ListRequest{
				TagFilter: "math",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Math Note", []string{"math"})
				createTestNote(repo, "Science Note", []string{"science"})
				createTestNote(repo, "History Note", []string{"history"})
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "filter by tag - multiple matches",
			request: ListRequest{
				TagFilter: "study",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{"study", "math"})
				createTestNote(repo, "Note 2", []string{"study", "science"})
				createTestNote(repo, "Note 3", []string{"history"})
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "filter by tag - no matches",
			request: ListRequest{
				TagFilter: "nonexistent",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{"math"})
				createTestNote(repo, "Note 2", []string{"science"})
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "filter by tag - case insensitive",
			request: ListRequest{
				TagFilter: "MATH",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{"math"})
				createTestNote(repo, "Note 2", []string{"science"})
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "sort by title ascending",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "title",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Zebra", []string{})
				createTestNote(repo, "Apple", []string{})
				createTestNote(repo, "Mango", []string{})
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "sort by title descending",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "title",
				Reverse:   true,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Zebra", []string{})
				createTestNote(repo, "Apple", []string{})
				createTestNote(repo, "Mango", []string{})
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "sort by date ascending",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "date",
				Reverse:   false,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{})
				createTestNote(repo, "Note 2", []string{})
				createTestNote(repo, "Note 3", []string{})
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "sort by date descending",
			request: ListRequest{
				TagFilter: "",
				SortBy:    "date",
				Reverse:   true,
			},
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{})
				createTestNote(repo, "Note 2", []string{})
				createTestNote(repo, "Note 3", []string{})
			},
			expectedCount: 3,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := mocks.NewMockRepository()

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Create service
			service := NewListService(mockRepo)

			// Execute
			ctx := context.Background()
			resp, err := service.Execute(ctx, tt.request)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if err != nil {
				return
			}

			// Verify count
			if resp.Total != tt.expectedCount {
				t.Errorf("Expected %d notes, got %d", tt.expectedCount, resp.Total)
			}

			if len(resp.Notes) != tt.expectedCount {
				t.Errorf("Expected %d notes in slice, got %d", tt.expectedCount, len(resp.Notes))
			}

			// Verify sorting
			if len(resp.Notes) > 1 {
				verifySorting(t, resp.Notes, tt.request.SortBy, tt.request.Reverse)
			}

			// Verify tag filter
			if tt.request.TagFilter != "" {
				for _, note := range resp.Notes {
					if !note.HasTag(tt.request.TagFilter) {
						t.Errorf("Note '%s' does not have tag '%s'", note.Title, tt.request.TagFilter)
					}
				}
			}
		})
	}
}

func TestListService_Search(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMocks     func(*mocks.MockRepository)
		expectedCount  int
		expectedTitles []string
	}{
		{
			name:  "search by title - exact match",
			query: "Graph Theory",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Graph Theory", []string{})
				createTestNote(repo, "Linear Algebra", []string{})
			},
			expectedCount:  1,
			expectedTitles: []string{"Graph Theory"},
		},
		{
			name:  "search by title - partial match",
			query: "graph",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Graph Theory", []string{})
				createTestNote(repo, "Graph Algorithms", []string{})
				createTestNote(repo, "Linear Algebra", []string{})
			},
			expectedCount:  2,
			expectedTitles: []string{"Graph Theory", "Graph Algorithms"},
		},
		{
			name:  "search by slug",
			query: "linear-algebra",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Linear Algebra", []string{})
				createTestNote(repo, "Graph Theory", []string{})
			},
			expectedCount:  1,
			expectedTitles: []string{"Linear Algebra"},
		},
		{
			name:  "search by tag",
			query: "math",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Calculus", []string{"math"})
				createTestNote(repo, "Physics", []string{"science"})
			},
			expectedCount:  1,
			expectedTitles: []string{"Calculus"},
		},
		{
			name:  "search with no results",
			query: "nonexistent",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{})
				createTestNote(repo, "Note 2", []string{})
			},
			expectedCount:  0,
			expectedTitles: []string{},
		},
		{
			name:  "empty query returns all",
			query: "",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Note 1", []string{})
				createTestNote(repo, "Note 2", []string{})
				createTestNote(repo, "Note 3", []string{})
			},
			expectedCount:  3,
			expectedTitles: []string{"Note 1", "Note 2", "Note 3"},
		},
		{
			name:  "case insensitive search",
			query: "GRAPH",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Graph Theory", []string{})
				createTestNote(repo, "Linear Algebra", []string{})
			},
			expectedCount:  1,
			expectedTitles: []string{"Graph Theory"},
		},
		{
			name:  "search matches multiple fields",
			query: "theory",
			setupMocks: func(repo *mocks.MockRepository) {
				createTestNote(repo, "Graph Theory", []string{})
				createTestNote(repo, "Set Theory", []string{})
				createTestNote(repo, "Number Theory Notes", []string{"theory"})
			},
			expectedCount:  3,
			expectedTitles: []string{"Graph Theory", "Set Theory", "Number Theory Notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := mocks.NewMockRepository()

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Create service
			service := NewListService(mockRepo)

			// Execute
			ctx := context.Background()
			req := SearchRequest{Query: tt.query}
			resp, err := service.Search(ctx, req)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify count
			if resp.Total != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, resp.Total)
			}

			if len(resp.Notes) != tt.expectedCount {
				t.Errorf("Expected %d notes in slice, got %d", tt.expectedCount, len(resp.Notes))
			}

			// Verify expected titles are present
			for _, expectedTitle := range tt.expectedTitles {
				found := false
				for _, note := range resp.Notes {
					if note.Title == expectedTitle {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find note with title '%s'", expectedTitle)
				}
			}
		})
	}
}

// Helper functions

func createTestNote(repo *mocks.MockRepository, title string, tags []string) {
	header, _ := domain.NewNoteHeader(title, tags)
	note := domain.NewNoteBody(header, "% test content")
	repo.Save(context.Background(), note)
}

func verifySorting(t *testing.T, notes []domain.NoteHeader, sortBy string, reverse bool) {
	t.Helper()

	for i := 0; i < len(notes)-1; i++ {
		switch sortBy {
		case "title":
			title1 := strings.ToLower(notes[i].Title)
			title2 := strings.ToLower(notes[i+1].Title)
			if reverse {
				if title1 < title2 {
					t.Errorf("Notes not sorted by title (reverse). '%s' should come after '%s'",
						notes[i].Title, notes[i+1].Title)
				}
			} else {
				if title1 > title2 {
					t.Errorf("Notes not sorted by title. '%s' should come before '%s'",
						notes[i+1].Title, notes[i].Title)
				}
			}
		case "date":
			date1 := notes[i].Date
			date2 := notes[i+1].Date
			if reverse {
				if date1 < date2 {
					t.Errorf("Notes not sorted by date (reverse). '%s' should come after '%s'",
						notes[i].Date, notes[i+1].Date)
				}
			} else {
				if date1 > date2 {
					t.Errorf("Notes not sorted by date. '%s' should come before '%s'",
						notes[i+1].Date, notes[i].Date)
				}
			}
		}
	}
}
