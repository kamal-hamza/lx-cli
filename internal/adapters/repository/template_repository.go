package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lx/internal/core/domain"
	"lx/pkg/vault"
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
