package repository

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/metadata"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// FileRepository implements the Repository port using the file system
type FileRepository struct {
	vault *vault.Vault
	mu    sync.RWMutex // Protects concurrent file operations
}

// NewFileRepository creates a new file-based repository
func NewFileRepository(vault *vault.Vault) *FileRepository {
	return &FileRepository{
		vault: vault,
	}
}

// ListHeaders returns all note headers by parsing file metadata
func (r *FileRepository) ListHeaders(ctx context.Context) ([]domain.NoteHeader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
			// Log warning but continue processing other files
			// This makes the system more robust - one bad file doesn't break everything
			continue
		}

		headers = append(headers, *header)
	}

	return headers, nil
}

// Save persists a note to the file system
func (r *FileRepository) Save(ctx context.Context, note *domain.NoteBody) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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
	r.mu.RLock()
	defer r.mu.RUnlock()

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
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, err := r.findFileBySlug(slug)
	return err == nil
}

// Delete removes a note by slug
func (r *FileRepository) Delete(ctx context.Context, slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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

// parseHeader extracts metadata from a LaTeX file using the robust metadata parser
func (r *FileRepository) parseHeader(filename string) (*domain.NoteHeader, error) {
	path := r.vault.GetNotePath(filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Use non-strict parser for reading existing files
	// This allows recovery from minor metadata issues
	meta, err := metadata.Extract(string(content))
	if err != nil {
		// Fallback: try to get basic info from filename
		return r.fallbackHeader(filename), nil
	}

	header := &domain.NoteHeader{
		Filename: filename,
		Slug:     domain.ParseFilename(filename),
		Title:    meta.Title,
		Date:     meta.Date,
		Tags:     meta.Tags,
	}

	// Ensure tags is never nil
	if header.Tags == nil {
		header.Tags = []string{}
	}

	// If no title found, use slug as fallback
	if header.Title == "" {
		header.Title = header.Slug
	}

	return header, nil
}

// fallbackHeader creates a minimal header when metadata parsing fails
func (r *FileRepository) fallbackHeader(filename string) *domain.NoteHeader {
	slug := domain.ParseFilename(filename)
	return &domain.NoteHeader{
		Filename: filename,
		Slug:     slug,
		Title:    slug,
		Date:     "",
		Tags:     []string{},
	}
}

// ensureMetadata ensures the note content has proper metadata comments
func (r *FileRepository) ensureMetadata(note *domain.NoteBody) string {
	content := note.Content

	// Try to extract existing metadata
	existingMeta, err := metadata.Extract(content)
	if err != nil || existingMeta == nil {
		// No valid metadata exists, create new one
		newMeta := &metadata.Metadata{
			Title: note.Header.Title,
			Date:  note.Header.Date,
			Tags:  note.Header.Tags,
		}
		return metadata.Update(content, newMeta)
	}

	// Update metadata with values from header
	// Header values take precedence over file metadata
	updatedMeta := &metadata.Metadata{
		Title: note.Header.Title,
		Date:  note.Header.Date,
		Tags:  note.Header.Tags,
	}

	// Use existing date if header doesn't specify one
	if updatedMeta.Date == "" && existingMeta.Date != "" {
		updatedMeta.Date = existingMeta.Date
	}

	// Merge tags if both exist
	if len(updatedMeta.Tags) == 0 && len(existingMeta.Tags) > 0 {
		updatedMeta.Tags = existingMeta.Tags
	}

	return metadata.Update(content, updatedMeta)
}

// findFileBySlug finds a file matching the given slug
// This is a helper that must be called with locks already held
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
	r.mu.Lock()
	defer r.mu.Unlock()

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
	_, existsErr := r.findFileBySlug(newSlug)
	if existsErr == nil && newSlug != oldSlug {
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

	// Read current content
	contentBytes, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read note: %w", err)
	}

	content := string(contentBytes)

	// Extract current metadata and update title
	existingMeta, err := metadata.Extract(content)
	if err != nil {
		// If metadata extraction fails, fall back to regex replacement
		titleRegex := regexp.MustCompile(`(?m)^%+\s*title:.*$`)
		content = titleRegex.ReplaceAllString(content, fmt.Sprintf("%%%% title: %s", newTitle))
	} else {
		// Update metadata properly
		existingMeta.Title = newTitle
		content = metadata.Update(content, existingMeta)
	}

	// Write updated content to new path
	if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write renamed note: %w", err)
	}

	// Remove old file only if the path changed
	if oldPath != newPath {
		if err := os.Remove(oldPath); err != nil {
			// Try to clean up the new file
			os.Remove(newPath)
			return fmt.Errorf("failed to remove old file: %w", err)
		}
	}

	return nil
}

// List returns all notes (full bodies)
func (r *FileRepository) List(ctx context.Context) ([]domain.NoteHeader, error) {
	return r.ListHeaders(ctx)
}
