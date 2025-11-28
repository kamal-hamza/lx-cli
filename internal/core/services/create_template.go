package services

import (
	"context"
	"fmt"
	"time"

	"lx/internal/core/domain"
	"lx/internal/core/ports"
)

type CreateTemplateService struct {
	templateRepo ports.TemplateRepository
}

func NewCreateTemplateService(templateRepo ports.TemplateRepository) *CreateTemplateService {
	return &CreateTemplateService{
		templateRepo: templateRepo,
	}
}

type CreateTemplateRequest struct {
	Title   string
	Content string
}

type CreateTemplateResponse struct {
	Template *domain.TemplateBody
	FilePath string
}

// Execute creates a new template with the given parameters
func (s *CreateTemplateService) Execute(ctx context.Context, req CreateTemplateRequest) (*CreateTemplateResponse, error) {
	// Validate title
	if req.Title == "" {
		return nil, fmt.Errorf("template title cannot be empty")
	}

	// Generate slug from title
	slug := domain.GenerateTemplateSlug(req.Title)
	if slug == "" {
		return nil, fmt.Errorf("failed to generate valid slug from title")
	}

	// Check if template already exists
	if s.templateRepo.Exists(ctx, slug) {
		return nil, fmt.Errorf("template with slug '%s' already exists", slug)
	}

	// Create template header
	header := domain.TemplateHeader{
		Title:    req.Title,
		Date:     time.Now().Format("2006-01-02"),
		Slug:     slug,
		Filename: slug + ".sty",
	}

	// Create template body
	template := &domain.TemplateBody{
		Header:  header,
		Content: req.Content,
	}

	// Save the template
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	return &CreateTemplateResponse{
		Template: template,
		FilePath: header.Filename,
	}, nil
}
