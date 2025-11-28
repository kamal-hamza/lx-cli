package services

import (
	"context"
	"testing"

	"lx/internal/core/domain"
	"lx/internal/core/ports/mocks"
)

func TestCreateTemplateService_Execute(t *testing.T) {
	tests := []struct {
		name        string
		request     CreateTemplateRequest
		setupMocks  func(*mocks.MockTemplateRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful template creation",
			request: CreateTemplateRequest{
				Title:   "homework-template",
				Content: "% LaTeX style file content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "template with hyphen",
			request: CreateTemplateRequest{
				Title:   "math-common",
				Content: "% Math template content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "template with underscore",
			request: CreateTemplateRequest{
				Title:   "hw_template",
				Content: "% Homework template",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "empty title should fail",
			request: CreateTemplateRequest{
				Title:   "",
				Content: "% Content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid template title",
		},
		{
			name: "title with spaces should fail",
			request: CreateTemplateRequest{
				Title:   "my template",
				Content: "% Content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid template title",
		},
		{
			name: "title with special characters should fail",
			request: CreateTemplateRequest{
				Title:   "template!@#",
				Content: "% Content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: true,
			errorMsg:    "invalid template title",
		},
		{
			name: "duplicate template should fail",
			request: CreateTemplateRequest{
				Title:   "existing-template",
				Content: "% New content",
			},
			setupMocks: func(tr *mocks.MockTemplateRepository) {
				// Create existing template
				template := &domain.TemplateBody{
					Header: domain.TemplateHeader{
						Title:    "existing-template",
						Slug:     "existing-template",
						Filename: "existing-template.sty",
						Date:     "2024-01-01",
					},
					Content: "% Existing content",
				}
				tr.Create(context.Background(), template)
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name: "template with empty content",
			request: CreateTemplateRequest{
				Title:   "empty-content",
				Content: "",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "template with alphanumeric name",
			request: CreateTemplateRequest{
				Title:   "template123",
				Content: "% Content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
		{
			name: "template with only numbers should work",
			request: CreateTemplateRequest{
				Title:   "12345",
				Content: "% Content",
			},
			setupMocks:  func(tr *mocks.MockTemplateRepository) {},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockTemplateRepo := mocks.NewMockTemplateRepository()

			// Setup mocks
			tt.setupMocks(mockTemplateRepo)

			// Create service
			service := NewCreateTemplateService(mockTemplateRepo)

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

			if resp.Template == nil {
				t.Error("Expected template in response but got nil")
				return
			}

			if resp.Template.Header.Title != tt.request.Title {
				t.Errorf("Expected title '%s', got '%s'", tt.request.Title, resp.Template.Header.Title)
			}

			if resp.Template.Content != tt.request.Content {
				t.Errorf("Expected content '%s', got '%s'", tt.request.Content, resp.Template.Content)
			}

			// Verify slug was generated
			if resp.Template.Header.Slug == "" {
				t.Error("Template slug is empty")
			}

			// Verify filename was generated
			if resp.Template.Header.Filename == "" {
				t.Error("Template filename is empty")
			}

			if resp.FilePath != resp.Template.Header.Filename {
				t.Errorf("FilePath '%s' does not match filename '%s'", resp.FilePath, resp.Template.Header.Filename)
			}

			// Verify template was saved
			if !mockTemplateRepo.Exists(ctx, resp.Template.Header.Slug) {
				t.Error("Template was not saved to repository")
			}
		})
	}
}

func TestCreateTemplateService_SlugGeneration(t *testing.T) {
	tests := []struct {
		title        string
		expectedSlug string
	}{
		{"homework", "homework"},
		{"math-common", "math-common"},
		{"hw_template", "hw_template"},
		{"MyTemplate", "mytemplate"},
		{"CAPS", "caps"},
		{"template123", "template123"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			mockTemplateRepo := mocks.NewMockTemplateRepository()
			service := NewCreateTemplateService(mockTemplateRepo)

			req := CreateTemplateRequest{
				Title:   tt.title,
				Content: "% Test content",
			}

			ctx := context.Background()
			resp, err := service.Execute(ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.Template.Header.Slug != tt.expectedSlug {
				t.Errorf("Expected slug '%s', got '%s'", tt.expectedSlug, resp.Template.Header.Slug)
			}
		})
	}
}

func TestCreateTemplateService_FilenameGeneration(t *testing.T) {
	mockTemplateRepo := mocks.NewMockTemplateRepository()
	service := NewCreateTemplateService(mockTemplateRepo)

	req := CreateTemplateRequest{
		Title:   "test-template",
		Content: "% Content",
	}

	ctx := context.Background()
	resp, err := service.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedFilename := "test-template.sty"
	if resp.Template.Header.Filename != expectedFilename {
		t.Errorf("Expected filename '%s', got '%s'", expectedFilename, resp.Template.Header.Filename)
	}

	if resp.FilePath != expectedFilename {
		t.Errorf("Expected FilePath '%s', got '%s'", expectedFilename, resp.FilePath)
	}
}

func TestCreateTemplateService_DateGeneration(t *testing.T) {
	mockTemplateRepo := mocks.NewMockTemplateRepository()
	service := NewCreateTemplateService(mockTemplateRepo)

	req := CreateTemplateRequest{
		Title:   "test-template",
		Content: "% Content",
	}

	ctx := context.Background()
	resp, err := service.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Template.Header.Date == "" {
		t.Error("Template date is empty")
	}

	// Date should be in YYYY-MM-DD format
	if len(resp.Template.Header.Date) != 10 {
		t.Errorf("Date format incorrect: %s", resp.Template.Header.Date)
	}
}

func TestCreateTemplateService_ContentPreservation(t *testing.T) {
	mockTemplateRepo := mocks.NewMockTemplateRepository()
	service := NewCreateTemplateService(mockTemplateRepo)

	originalContent := `% My Custom Template
\ProvidesPackage{mytemplate}
\RequirePackage{amsmath}
\RequirePackage{graphicx}

\newcommand{\customcmd}{Custom Command}
`

	req := CreateTemplateRequest{
		Title:   "custom-template",
		Content: originalContent,
	}

	ctx := context.Background()
	resp, err := service.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Template.Content != originalContent {
		t.Error("Template content was modified")
	}

	// Verify it was saved correctly
	savedTemplate, err := mockTemplateRepo.Get(ctx, resp.Template.Header.Slug)
	if err != nil {
		t.Fatalf("Failed to retrieve saved template: %v", err)
	}

	// The mock returns a Template struct, not TemplateBody, so we can't directly check content
	// But we verified it exists
	if savedTemplate == nil {
		t.Error("Saved template is nil")
	}
}

func TestCreateTemplateService_MultipleTemplates(t *testing.T) {
	mockTemplateRepo := mocks.NewMockTemplateRepository()
	service := NewCreateTemplateService(mockTemplateRepo)

	ctx := context.Background()
	templates := []string{"template1", "template2", "template3"}

	for _, title := range templates {
		req := CreateTemplateRequest{
			Title:   title,
			Content: "% " + title + " content",
		}

		_, err := service.Execute(ctx, req)
		if err != nil {
			t.Errorf("Failed to create template '%s': %v", title, err)
		}
	}

	// Verify all templates exist
	allTemplates, err := mockTemplateRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(allTemplates) != len(templates) {
		t.Errorf("Expected %d templates, got %d", len(templates), len(allTemplates))
	}
}
