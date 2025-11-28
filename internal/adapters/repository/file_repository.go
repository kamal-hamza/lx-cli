package repository

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// FileRepository implements the Repository port using the file system
type FileRepository struct {
	vault *vault.Vault
}

// NewFileRepository creates a new file-based repository
func NewFileRepository(vault *vault.Vault) *FileRepository {
	return &FileRepository{
		vault: vault,
	}
}

// ListHeaders returns all note headers by parsing file metadata
func (r *FileRepository) ListHeaders(ctx context.Context) ([]domain.NoteHeader, error) {
	var headers []domain.NoteHeader

	entries, err := os.ReadDir(r.vault.NotesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return headers, nil
		}
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tex") {
			continue
		}

		header, err := r.parseHeader(entry.Name())
		if err != nil {
			// Skip malformed files but don't fail the entire operation
			continue
		}

		headers = append(headers, *header)
	}

	return headers, nil
}

// Save persists a note to the file system
func (r *FileRepository) Save(ctx context.Context, note *domain.NoteBody) error {
	path := r.vault.GetNotePath(note.Header.Filename)

	// Ensure content has proper metadata
	content := r.ensureMetadata(note)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save note: %w", err)
	}

	return nil
}

// Get retrieves a note by slug
func (r *FileRepository) Get(ctx context.Context, slug string) (*domain.NoteBody, error) {
	// Find the file with this slug
	filename, err := r.findFileBySlug(slug)
	if err != nil {
		return nil, err
	}

	path := r.vault.GetNotePath(filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read note: %w", err)
	}

	header, err := r.parseHeader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	return &domain.NoteBody{
		Header:  *header,
		Content: string(content),
	}, nil
}

// Exists checks if a note with the given slug exists
func (r *FileRepository) Exists(ctx context.Context, slug string) bool {
	_, err := r.findFileBySlug(slug)
	return err == nil
}

// Delete removes a note by slug
func (r *FileRepository) Delete(ctx context.Context, slug string) error {
	filename, err := r.findFileBySlug(slug)
	if err != nil {
		return err
	}

	path := r.vault.GetNotePath(filename)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// parseHeader extracts metadata from a LaTeX file
func (r *FileRepository) parseHeader(filename string) (*domain.NoteHeader, error) {
	path := r.vault.GetNotePath(filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	header := &domain.NoteHeader{
		Filename: filename,
		Slug:     domain.ParseFilename(filename),
		Tags:     []string{},
	}

	text := string(content)

	// Parse title from % title: format
	titleRe := regexp.MustCompile(`(?m)^%+\s*title:\s*(.+)$`)
	if matches := titleRe.FindStringSubmatch(text); len(matches) > 1 {
		header.Title = strings.TrimSpace(matches[1])
	}

	// Parse date from % date: format
	dateRe := regexp.MustCompile(`(?m)^%+\s*date:\s*(.+)$`)
	if matches := dateRe.FindStringSubmatch(text); len(matches) > 1 {
		header.Date = strings.TrimSpace(matches[1])
	}

	// Parse tags from % tags: format (comma-separated)
	tagsRe := regexp.MustCompile(`(?m)^%+\s*tags:\s*(.*)$`)
	if matches := tagsRe.FindStringSubmatch(text); len(matches) > 1 {
		tagsStr := strings.TrimSpace(matches[1])
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					header.Tags = append(header.Tags, trimmed)
				}
			}
		}
	}

	// If no title found, use filename
	if header.Title == "" {
		header.Title = header.Slug
	}

	return header, nil
}

// ensureMetadata ensures the note content has proper metadata comments
func (r *FileRepository) ensureMetadata(note *domain.NoteBody) string {
	content := note.Content

	// Check if metadata already exists
	hasMetadata := strings.Contains(content, "% title:")

	if !hasMetadata {
		// Prepend metadata
		metadata := fmt.Sprintf("%% Metadata\n")
		metadata += fmt.Sprintf("%% title: %s\n", note.Header.Title)
		metadata += fmt.Sprintf("%% date: %s\n", note.Header.Date)
		if len(note.Header.Tags) > 0 {
			metadata += fmt.Sprintf("%% tags: %s\n", strings.Join(note.Header.Tags, ", "))
		} else {
			metadata += "%% tags: \n"
		}
		metadata += "\n"

		content = metadata + content
	}

	return content
}

// findFileBySlug finds a file matching the given slug
func (r *FileRepository) findFileBySlug(slug string) (string, error) {
	entries, err := os.ReadDir(r.vault.NotesPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tex") {
			continue
		}

		fileSlug := domain.ParseFilename(entry.Name())
		if fileSlug == slug {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("note not found: %s", slug)
}

// FindByQuery searches for notes matching a fuzzy query
func (r *FileRepository) FindByQuery(ctx context.Context, query string) ([]domain.NoteHeader, error) {
	headers, err := r.ListHeaders(ctx)
	if err != nil {
		return nil, err
	}

	if query == "" {
		return headers, nil
	}

	query = strings.ToLower(query)
	var matches []domain.NoteHeader

	for _, header := range headers {
		// Check title match
		if strings.Contains(strings.ToLower(header.Title), query) {
			matches = append(matches, header)
			continue
		}

		// Check slug match
		if strings.Contains(strings.ToLower(header.Slug), query) {
			matches = append(matches, header)
			continue
		}

		// Check tag match
		for _, tag := range header.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				matches = append(matches, header)
				break
			}
		}
	}

	return matches, nil
}

// Rename renames a note from oldSlug to newTitle
func (r *FileRepository) Rename(ctx context.Context, oldSlug string, newTitle string) error {
	// Validate new title
	if err := domain.ValidateTitle(newTitle); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}

	// Find the old file
	oldFilename, err := r.findFileBySlug(oldSlug)
	if err != nil {
		return fmt.Errorf("note not found: %w", err)
	}

	// Generate new slug and filename
	newSlug := domain.GenerateSlug(newTitle)

	// Check if new slug already exists
	if r.Exists(ctx, newSlug) && newSlug != oldSlug {
		return fmt.Errorf("note with slug '%s' already exists", newSlug)
	}

	// Preserve date prefix if present
	newFilename := domain.GenerateFilename(newSlug)
	if strings.Contains(oldFilename, "-") {
		parts := strings.SplitN(oldFilename, "-", 2)
		if len(parts) == 2 && len(parts[0]) == 8 { // YYYYMMDD check
			newFilename = parts[0] + "-" + newSlug + ".tex"
		}
	}

	oldPath := r.vault.GetNotePath(oldFilename)
	newPath := r.vault.GetNotePath(newFilename)

	// Read and update content
	contentBytes, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read note: %w", err)
	}

	content := string(contentBytes)

	// Update title in metadata
	titleRegex := regexp.MustCompile(`(?m)^%+\s*title:.*$`)
	content = titleRegex.ReplaceAllString(content, fmt.Sprintf("%%%% title: %s", newTitle))

	// Write updated content to old path first
	if err := os.WriteFile(oldPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update note content: %w", err)
	}

	// Rename the file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// List returns all notes (full bodies)
func (r *FileRepository) List(ctx context.Context) ([]domain.NoteHeader, error) {
	return r.ListHeaders(ctx)
}
