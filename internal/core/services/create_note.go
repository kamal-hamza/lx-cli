package services

import (
	"context"
	"fmt"
	"strings"

	"lx/internal/core/domain"
	"lx/internal/core/ports"
)

// CreateNoteService handles the creation of new notes
type CreateNoteService struct {
	noteRepo     ports.Repository
	templateRepo ports.TemplateRepository
}

// NewCreateNoteService creates a new note creation service
func NewCreateNoteService(noteRepo ports.Repository, templateRepo ports.TemplateRepository) *CreateNoteService {
	return &CreateNoteService{
		noteRepo:     noteRepo,
		templateRepo: templateRepo,
	}
}

// CreateNoteRequest represents a request to create a new note
type CreateNoteRequest struct {
	Title        string
	Tags         []string
	TemplateName string
}

// CreateNoteResponse represents the response from creating a note
type CreateNoteResponse struct {
	Note     *domain.NoteBody
	FilePath string
}

// Execute creates a new note with the given parameters
func (s *CreateNoteService) Execute(ctx context.Context, req CreateNoteRequest) (*CreateNoteResponse, error) {
	// Validate title
	if err := domain.ValidateTitle(req.Title); err != nil {
		return nil, fmt.Errorf("invalid title: %w", err)
	}

	// Create note header
	header, err := domain.NewNoteHeader(req.Title, req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create note header: %w", err)
	}

	// Check if note already exists
	if s.noteRepo.Exists(ctx, header.Slug) {
		return nil, fmt.Errorf("note with slug '%s' already exists", header.Slug)
	}

	// Render content based on template
	content, err := s.renderContent(ctx, req.TemplateName, header)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	// Create note body
	note := domain.NewNoteBody(header, content)

	// Save the note
	if err := s.noteRepo.Save(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to save note: %w", err)
	}

	return &CreateNoteResponse{
		Note:     note,
		FilePath: header.Filename,
	}, nil
}

// renderContent generates the initial LaTeX content for the note
func (s *CreateNoteService) renderContent(ctx context.Context, templateName string, header *domain.NoteHeader) (string, error) {
	var builder strings.Builder

	// Start with metadata comments
	builder.WriteString("%%%% Metadata\n")
	builder.WriteString(fmt.Sprintf("%% title: %s\n", header.Title))
	builder.WriteString(fmt.Sprintf("%% date: %s\n", header.Date))
	if len(header.Tags) > 0 {
		builder.WriteString(fmt.Sprintf("%% tags: %s\n", strings.Join(header.Tags, ", ")))
	}
	builder.WriteString("\n")

	// Document class
	builder.WriteString("\\documentclass[12pt]{article}\n\n")

	// If template is specified, include it
	if templateName != "" {
		// Validate template exists
		if !s.templateRepo.Exists(ctx, templateName) {
			return "", fmt.Errorf("template '%s' not found", templateName)
		}
		builder.WriteString(fmt.Sprintf("\\usepackage{%s}\n", templateName))
	}

	// Common packages
	builder.WriteString("\\usepackage[utf8]{inputenc}\n")
	builder.WriteString("\\usepackage[T1]{fontenc}\n")
	builder.WriteString("\\usepackage{amsmath}\n")
	builder.WriteString("\\usepackage{amssymb}\n")
	builder.WriteString("\\usepackage{geometry}\n")
	builder.WriteString("\\geometry{margin=1in}\n\n")

	// Title and author
	builder.WriteString(fmt.Sprintf("\\title{%s}\n", header.Title))
	builder.WriteString(fmt.Sprintf("\\date{%s}\n\n", header.Date))

	// Begin document
	builder.WriteString("\\begin{document}\n\n")
	builder.WriteString("\\maketitle\n\n")

	// Content placeholder
	builder.WriteString("% Your notes go here\n\n")

	// End document
	builder.WriteString("\\end{document}\n")

	return builder.String(), nil
}
