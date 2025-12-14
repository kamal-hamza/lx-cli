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
	vault             *vault.Vault
	customTemplateDir string
}

// NewTemplateRepository creates a new file-based template repository
func NewTemplateRepository(vault *vault.Vault, customTemplateDir string) *TemplateRepository {
	return &TemplateRepository{
		vault:             vault,
		customTemplateDir: customTemplateDir,
	}
}

// List returns all available templates (including custom ones)
func (r *TemplateRepository) List(ctx context.Context) ([]domain.Template, error) {
	var templates []domain.Template

	// 1. Scan Vault Templates
	vaultTemplates, err := r.scanDirectory(r.vault.TemplatesPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}
	templates = append(templates, vaultTemplates...)

	// 2. Scan Custom Directory if set
	if r.customTemplateDir != "" {
		dir := r.expandPath(r.customTemplateDir)
		customTemplates, err := r.scanDirectory(dir)
		if err == nil {
			templates = append(templates, customTemplates...)
		}
	}

	return templates, nil
}

func (r *TemplateRepository) scanDirectory(dir string) ([]domain.Template, error) {
	var templates []domain.Template
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sty") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".sty")
		templates = append(templates, domain.Template{
			Name: name,
			Path: filepath.Join(dir, entry.Name()),
		})
	}
	return templates, nil
}

func (r *TemplateRepository) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// Exists checks if a template with the given name exists
func (r *TemplateRepository) Exists(ctx context.Context, name string) bool {
	_, err := r.Get(ctx, name)
	return err == nil
}

// Get retrieves a template by name
func (r *TemplateRepository) Get(ctx context.Context, name string) (*domain.Template, error) {
	filename := name
	if !strings.HasSuffix(filename, ".sty") {
		filename = name + ".sty"
	}

	// Check Vault
	path := r.vault.GetTemplatePath(filename)
	if _, err := os.Stat(path); err == nil {
		return &domain.Template{
			Name: strings.TrimSuffix(filepath.Base(filename), ".sty"),
			Path: path,
		}, nil
	}

	// Check Custom Dir
	if r.customTemplateDir != "" {
		dir := r.expandPath(r.customTemplateDir)
		path = filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return &domain.Template{
				Name: strings.TrimSuffix(filepath.Base(filename), ".sty"),
				Path: path,
			}, nil
		}
	}

	return nil, fmt.Errorf("template not found: %s", name)
}

// Create creates a new template file (Always creates in Vault)
func (r *TemplateRepository) Create(ctx context.Context, template *domain.TemplateBody) error {
	// Generate filename from slug
	filename := template.Header.Slug + ".sty"
	path := r.vault.GetTemplatePath(filename)

	// Check if template already exists
	if r.Exists(ctx, template.Header.Slug) {
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
