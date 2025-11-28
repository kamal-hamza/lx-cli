package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// TemplateRepository implements the TemplateRepository port using the file system
type TemplateRepository struct {
	vault *vault.Vault
}

// NewTemplateRepository creates a new file-based template repository
func NewTemplateRepository(vault *vault.Vault) *TemplateRepository {
	return &TemplateRepository{
		vault: vault,
	}
}

// List returns all available templates
func (r *TemplateRepository) List(ctx context.Context) ([]domain.Template, error) {
	var templates []domain.Template

	entries, err := os.ReadDir(r.vault.TemplatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return templates, nil
		}
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sty") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".sty")
		path := r.vault.GetTemplatePath(entry.Name())

		templates = append(templates, domain.Template{
			Name: name,
			Path: path,
		})
	}

	return templates, nil
}

// Exists checks if a template with the given name exists
func (r *TemplateRepository) Exists(ctx context.Context, name string) bool {
	filename := name
	if !strings.HasSuffix(filename, ".sty") {
		filename = name + ".sty"
	}

	path := r.vault.GetTemplatePath(filename)
	_, err := os.Stat(path)
	return err == nil
}

// Get retrieves a template by name
func (r *TemplateRepository) Get(ctx context.Context, name string) (*domain.Template, error) {
	filename := name
	if !strings.HasSuffix(filename, ".sty") {
		filename = name + ".sty"
	}

	path := r.vault.GetTemplatePath(filename)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("template not found: %s", name)
		}
		return nil, fmt.Errorf("failed to access template: %w", err)
	}

	return &domain.Template{
		Name: strings.TrimSuffix(filepath.Base(filename), ".sty"),
		Path: path,
	}, nil
}

// Create creates a new template file
func (r *TemplateRepository) Create(ctx context.Context, template *domain.TemplateBody) error {
	// Generate filename from slug
	filename := template.Header.Slug + ".sty"
	path := r.vault.GetTemplatePath(filename)

	// Check if template already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("template already exists: %s", template.Header.Slug)
	}

	// Create the template content
	content := r.renderTemplateContent(template)

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	return nil
}

// renderTemplateContent generates the LaTeX package content for a template
func (r *TemplateRepository) renderTemplateContent(template *domain.TemplateBody) string {
	var builder strings.Builder

	// Package identification
	builder.WriteString("\\NeedsTeXFormat{LaTeX2e}\n")
	builder.WriteString(fmt.Sprintf("\\ProvidesPackage{%s}[%s %s]\n\n",
		template.Header.Slug,
		time.Now().Format("2006/01/02"),
		template.Header.Title))

	// Add custom content if provided
	if template.Content != "" {
		builder.WriteString(template.Content)
		builder.WriteString("\n")
	} else {
		// Default template structure
		builder.WriteString("% Template packages and settings\n")
		builder.WriteString("% Add your custom LaTeX package imports and configurations here\n\n")
		builder.WriteString("% Example:\n")
		builder.WriteString("% \\RequirePackage{graphicx}\n")
		builder.WriteString("% \\RequirePackage{hyperref}\n\n")
	}

	builder.WriteString("\\endinput\n")

	return builder.String()
}
