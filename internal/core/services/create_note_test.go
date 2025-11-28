package services

import (
	"context"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
)

func TestCreateNoteService_Execute(t *testing.T) {
	tests := []struct {
		name        string
		request     CreateNoteRequest
		setupMocks  func(*mocks.MockRepository, *mocks.MockTemplateRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful note creation",
			request: CreateNoteRequest{
				Title:        "Test Note",
				Tags:         []string{"test", "example"},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "note creation with template",
			request: CreateNoteRequest{
				Title:        "Note with Template",
				Tags:         []string{"homework"},
				TemplateName: "homework-template",
			},
			setupMocks: func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {
				// Create a template first
				template := &domain.TemplateBody{
					Header: domain.TemplateHeader{
						Title:    "Homework Template",
						Slug:     "homework-template",
						Filename: "homework-template.sty",
						Date:     "2024-01-01",
					},
					Content: "% Homework template content",
				}
				tr.Create(context.Background(), template)
			},
			expectError: false,
		},
		{
			name: "empty title should fail",
			request: CreateNoteRequest{
				Title:        "",
				Tags:         []string{},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid title",
		},
		{
			name: "whitespace only title should fail",
			request: CreateNoteRequest{
				Title:        "   ",
				Tags:         []string{},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid title",
		},
		{
			name: "very long title should fail",
			request: CreateNoteRequest{
				Title:        string(make([]byte, 201)),
				Tags:         []string{},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid title",
		},
		{
			name: "duplicate note should fail",
			request: CreateNoteRequest{
				Title:        "Duplicate Note",
				Tags:         []string{},
				TemplateName: "",
			},
			setupMocks: func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {
				// Create a note first
				header, _ := domain.NewNoteHeader("Duplicate Note", []string{})
				note := domain.NewNoteBody(header, "% existing content")
				nr.Save(context.Background(), note)
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name: "non-existent template should fail",
			request: CreateNoteRequest{
				Title:        "Note with Missing Template",
				Tags:         []string{},
				TemplateName: "nonexistent-template",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "note with no tags",
			request: CreateNoteRequest{
				Title:        "Tagless Note",
				Tags:         []string{},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "note with multiple tags",
			request: CreateNoteRequest{
				Title:        "Multi-Tag Note",
				Tags:         []string{"math", "science", "homework", "final"},
				TemplateName: "",
			},
			setupMocks:  func(nr *mocks.MockRepository, tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repositories
			mockNoteRepo := mocks.NewMockRepository()
			mockTemplateRepo := mocks.NewMockTemplateRepository()

			// Setup mocks
			tt.setupMocks(mockNoteRepo, mockTemplateRepo)

			// Create service
			service := NewCreateNoteService(mockNoteRepo, mockTemplateRepo)

			// Execute
			ctx := context.Background()
			resp, err := service.Execute(ctx, tt.request)

			// Verify error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify response
			if resp == nil {
				t.Error("Expected response but got nil")
				return
			}

			if resp.Note == nil {
				t.Error("Expected note in response but got nil")
				return
			}

			if resp.Note.Header.Title != tt.request.Title {
				t.Errorf("Expected title '%s', got '%s'", tt.request.Title, resp.Note.Header.Title)
			}

			if len(resp.Note.Header.Tags) != len(tt.request.Tags) {
				t.Errorf("Expected %d tags, got %d", len(tt.request.Tags), len(resp.Note.Header.Tags))
			}

			// Verify note was saved
			slug := domain.GenerateSlug(tt.request.Title)
			if !mockNoteRepo.Exists(ctx, slug) {
				t.Error("Note was not saved to repository")
			}

			// Verify content contains metadata
			savedNote, _ := mockNoteRepo.Get(ctx, slug)
			if savedNote == nil {
				t.Error("Could not retrieve saved note")
				return
			}

			if !contains(savedNote.Content, "% title:") {
				t.Error("Note content missing title metadata")
			}

			if !contains(savedNote.Content, "\\documentclass") {
				t.Error("Note content missing document class")
			}

			// Verify template is included if specified
			if tt.request.TemplateName != "" && !tt.expectError {
				if !contains(savedNote.Content, "\\usepackage{"+tt.request.TemplateName+"}") {
					t.Errorf("Note content missing template package: %s", tt.request.TemplateName)
				}
			}
		})
	}
}

func TestCreateNoteService_SlugGeneration(t *testing.T) {
	tests := []struct {
		title        string
		expectedSlug string
	}{
		{"Simple Title", "simple-title"},
		{"Title with Numbers 123", "title-with-numbers-123"},
		{"Title with Special!@# Characters", "title-with-special-characters"},
		{"Multiple    Spaces", "multiple-spaces"},
		{"  Leading and Trailing  ", "leading-and-trailing"},
		{"CamelCaseTitle", "camelcasetitle"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			mockNoteRepo := mocks.NewMockRepository()
			mockTemplateRepo := mocks.NewMockTemplateRepository()
			service := NewCreateNoteService(mockNoteRepo, mockTemplateRepo)

			req := CreateNoteRequest{
				Title:        tt.title,
				Tags:         []string{},
				TemplateName: "",
			}

			ctx := context.Background()
			resp, err := service.Execute(ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.Note.Header.Slug != tt.expectedSlug {
				t.Errorf("Expected slug '%s', got '%s'", tt.expectedSlug, resp.Note.Header.Slug)
			}
		})
	}
}

func TestCreateNoteService_ContentGeneration(t *testing.T) {
	mockNoteRepo := mocks.NewMockRepository()
	mockTemplateRepo := mocks.NewMockTemplateRepository()
	service := NewCreateNoteService(mockNoteRepo, mockTemplateRepo)

	req := CreateNoteRequest{
		Title:        "Test Content Generation",
		Tags:         []string{"test", "content"},
		TemplateName: "",
	}

	ctx := context.Background()
	resp, err := service.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content := resp.Note.Content

	// Verify required LaTeX elements
	requiredElements := []string{
		"% title: Test Content Generation",
		"% tags: test, content",
		"\\documentclass",
		"\\usepackage{amsmath}",
		"\\usepackage{amssymb}",
		"\\title{Test Content Generation}",
		"\\begin{document}",
		"\\maketitle",
		"\\end{document}",
	}

	for _, elem := range requiredElements {
		if !contains(content, elem) {
			t.Errorf("Content missing required element: %s", elem)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
