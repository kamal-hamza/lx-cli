package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports"
	"github.com/kamal-hamza/lx-cli/pkg/metadata"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

type FileRepository struct {
	vault *vault.Vault
	mu    sync.RWMutex
}

// NewFileRepository creates a new file-based repository
func NewFileRepository(vault *vault.Vault) *FileRepository {
	return &FileRepository{
		vault: vault,
	}
}

// Ensure it implements the interface
var _ ports.Repository = (*FileRepository)(nil)

// ListHeaders returns all note headers
func (r *FileRepository) ListHeaders(ctx context.Context) ([]domain.NoteHeader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.vault.NotesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	var notes []domain.NoteHeader
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".tex" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Read the file header
		path := filepath.Join(r.vault.NotesPath, entry.Name())
		header, err := r.readHeader(path, entry.Name(), info)
		if err != nil {
			continue
		}
		notes = append(notes, *header)
	}

	return notes, nil
}

// Get retrieves a note by its slug
func (r *FileRepository) Get(ctx context.Context, slug string) (*domain.NoteBody, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Find the file associated with the slug
	filename, err := r.findFilenameBySlug(slug)
	if err != nil {
		return nil, err
	}

	// 2. Read file content
	path := filepath.Join(r.vault.NotesPath, filename)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// 3. Extract metadata using Extract (formerly Parse)
	meta, err := metadata.Extract(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// 4. Construct domain objects
	header := domain.NoteHeader{
		Title:    meta.Title,
		Date:     meta.Date,
		Tags:     meta.Tags,
		Slug:     slug,
		Filename: filename,
	}

	return &domain.NoteBody{
		Header:  header,
		Content: content,
	}, nil
}

// Save writes a note to disk
func (r *FileRepository) Save(ctx context.Context, note *domain.NoteBody) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	path := filepath.Join(r.vault.NotesPath, note.Header.Filename)
	return os.WriteFile(path, []byte(note.Content), 0644)
}

// Delete removes a note
func (r *FileRepository) Delete(ctx context.Context, slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filename, err := r.findFilenameBySlug(slug)
	if err != nil {
		return err
	}

	path := filepath.Join(r.vault.NotesPath, filename)
	return os.Remove(path)
}

// Exists checks if a slug exists
func (r *FileRepository) Exists(ctx context.Context, slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, err := r.findFilenameBySlug(slug)
	return err == nil
}

// Rename updates a note's title and filename
func (r *FileRepository) Rename(ctx context.Context, oldSlug string, newTitle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Find existing file
	oldFilename, err := r.findFilenameBySlug(oldSlug)
	if err != nil {
		return err
	}

	// 2. Read existing content
	oldPath := filepath.Join(r.vault.NotesPath, oldFilename)
	contentBytes, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read original file: %w", err)
	}
	content := string(contentBytes)

	// 3. Generate new slug
	newSlug := domain.GenerateSlug(newTitle)
	if newSlug == oldSlug {
		// Just update title in metadata, keep filename
		newContent, err := metadata.UpdateTitle(content, newTitle)
		if err != nil {
			return err
		}
		return os.WriteFile(oldPath, []byte(newContent), 0644)
	}

	// 4. Generate new filename (PRESERVING DATE)
	newFilename := preserveDatePrefix(oldFilename, newSlug)

	// Ensure new filename doesn't already exist
	newPath := filepath.Join(r.vault.NotesPath, newFilename)
	if _, err := os.Stat(newPath); err == nil && newFilename != oldFilename {
		return fmt.Errorf("destination filename already exists: %s", newFilename)
	}

	// 5. Update content metadata
	newContent, err := metadata.UpdateTitle(content, newTitle)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// 6. Write new file
	if err := os.WriteFile(newPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	// 7. Remove old file
	if err := os.Remove(oldPath); err != nil {
		return fmt.Errorf("failed to remove old file: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func (r *FileRepository) findFilenameBySlug(targetSlug string) (string, error) {
	entries, err := os.ReadDir(r.vault.NotesPath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".tex" {
			slug := domain.ParseFilename(entry.Name())
			if slug == targetSlug {
				return entry.Name(), nil
			}
		}
	}

	return "", fmt.Errorf("note not found: %s", targetSlug)
}

func (r *FileRepository) readHeader(path string, filename string, info os.FileInfo) (*domain.NoteHeader, error) {
	// Read first 1KB for metadata to be fast
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	content := string(buf[:n])

	// Use Extract instead of Parse
	meta, err := metadata.Extract(content)
	if err != nil {
		// Fallback for files without valid metadata
		return &domain.NoteHeader{
			Title:    domain.ParseFilename(filename),
			Date:     info.ModTime().Format("2006-01-02"),
			Slug:     domain.ParseFilename(filename),
			Filename: filename,
		}, nil
	}

	return &domain.NoteHeader{
		Title:    meta.Title,
		Date:     meta.Date,
		Tags:     meta.Tags,
		Slug:     domain.ParseFilename(filename),
		Filename: filename,
	}, nil
}

func preserveDatePrefix(oldFilename, newSlug string) string {
	oldName := strings.TrimSuffix(oldFilename, ".tex")
	re := regexp.MustCompile(`^(\d{8}|\d{4}-\d{2}-\d{2})-(.+)$`)
	matches := re.FindStringSubmatch(oldName)

	if len(matches) == 3 {
		datePart := matches[1]
		return fmt.Sprintf("%s-%s.tex", datePart, newSlug)
	}
	return fmt.Sprintf("%s.tex", newSlug)
}
