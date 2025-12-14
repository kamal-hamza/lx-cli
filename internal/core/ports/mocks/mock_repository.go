package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	mu      sync.RWMutex
	notes   map[string]*domain.NoteBody
	headers map[string]*domain.NoteHeader
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		notes:   make(map[string]*domain.NoteBody),
		headers: make(map[string]*domain.NoteHeader),
	}
}

// ListHeaders returns all note headers
func (m *MockRepository) ListHeaders(ctx context.Context) ([]domain.NoteHeader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	headers := make([]domain.NoteHeader, 0, len(m.headers))
	for _, h := range m.headers {
		headers = append(headers, *h)
	}
	return headers, nil
}

// Save persists a note to storage
func (m *MockRepository) Save(ctx context.Context, note *domain.NoteBody) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notes[note.Header.Slug] = note
	m.headers[note.Header.Slug] = &note.Header
	return nil
}

// Get retrieves a note by slug
// This was likely the missing method causing your errors
func (m *MockRepository) Get(ctx context.Context, slug string) (*domain.NoteBody, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	note, ok := m.notes[slug]
	if !ok {
		return nil, fmt.Errorf("note not found: %s", slug)
	}
	return note, nil
}

// Exists checks if a note with the given slug exists
func (m *MockRepository) Exists(ctx context.Context, slug string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.notes[slug]
	return ok
}

// Delete removes a note by slug
func (m *MockRepository) Delete(ctx context.Context, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.notes[slug]; !ok {
		return fmt.Errorf("note not found: %s", slug)
	}

	delete(m.notes, slug)
	delete(m.headers, slug)
	return nil
}

// Rename renames a note from oldSlug to newTitle
func (m *MockRepository) Rename(ctx context.Context, oldSlug string, newTitle string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, ok := m.notes[oldSlug]
	if !ok {
		return fmt.Errorf("note not found: %s", oldSlug)
	}

	if err := domain.ValidateTitle(newTitle); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}

	newSlug := domain.GenerateSlug(newTitle)

	if _, exists := m.notes[newSlug]; exists && newSlug != oldSlug {
		return fmt.Errorf("note with slug '%s' already exists", newSlug)
	}

	note.Header.Title = newTitle
	note.Header.Slug = newSlug
	note.Header.Filename = domain.GenerateFilename(newSlug)

	if newSlug != oldSlug {
		delete(m.notes, oldSlug)
		delete(m.headers, oldSlug)
	}

	m.notes[newSlug] = note
	m.headers[newSlug] = &note.Header

	return nil
}

// --- MockTemplateRepository ---

type MockTemplateRepository struct {
	mu        sync.RWMutex
	templates map[string]*domain.TemplateBody
}

func NewMockTemplateRepository() *MockTemplateRepository {
	return &MockTemplateRepository{
		templates: make(map[string]*domain.TemplateBody),
	}
}

func (m *MockTemplateRepository) List(ctx context.Context) ([]domain.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]domain.Template, 0, len(m.templates))
	for _, t := range m.templates {
		templates = append(templates, domain.Template{
			Name: t.Header.Slug,
			Path: "/mock/path/" + t.Header.Filename,
		})
	}
	return templates, nil
}

func (m *MockTemplateRepository) Exists(ctx context.Context, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.templates[name]
	return ok
}

func (m *MockTemplateRepository) Get(ctx context.Context, name string) (*domain.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	template, ok := m.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return &domain.Template{
		Name: template.Header.Slug,
		Path: "/mock/path/" + template.Header.Filename,
	}, nil
}

func (m *MockTemplateRepository) Create(ctx context.Context, template *domain.TemplateBody) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.templates[template.Header.Slug]; ok {
		return fmt.Errorf("template already exists: %s", template.Header.Slug)
	}
	m.templates[template.Header.Slug] = template
	return nil
}

// --- MockCompiler ---

type MockCompiler struct {
	mu           sync.Mutex
	calls        []string
	shouldFail   bool
	failError    error
	outputPrefix string
}

func NewMockCompiler() *MockCompiler {
	return &MockCompiler{
		outputPrefix: "/fake/cache/",
	}
}

func (m *MockCompiler) Compile(ctx context.Context, inputPath string, env []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, inputPath)
	if m.shouldFail {
		if m.failError != nil {
			return m.failError
		}
		return fmt.Errorf("compile failed for %s", inputPath)
	}
	return nil
}

func (m *MockCompiler) GetOutputPath(slug string) string {
	return m.outputPrefix + slug + ".pdf"
}

func (m *MockCompiler) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failError = err
}

func (m *MockCompiler) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

func (m *MockCompiler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.shouldFail = false
	m.failError = nil
}

// --- MockPreprocessor ---

type MockPreprocessor struct {
	mu         sync.Mutex
	calls      []string
	shouldFail bool
	failError  error
	mockPath   string
}

func NewMockPreprocessor() *MockPreprocessor {
	return &MockPreprocessor{
		mockPath: "/fake/cache/mock_preprocessed.tex",
	}
}

func (m *MockPreprocessor) Process(slug string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, slug)
	if m.shouldFail {
		if m.failError != nil {
			return "", m.failError
		}
		return "", fmt.Errorf("preprocessing failed for %s", slug)
	}
	return m.mockPath, nil
}

func (m *MockPreprocessor) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failError = err
}

func (m *MockPreprocessor) SetMockPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockPath = path
}

func (m *MockPreprocessor) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

func (m *MockPreprocessor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.shouldFail = false
	m.failError = nil
}

// --- MockAssetRepository (For Assets) ---

// MockAssetRepository is a mock implementation of the AssetRepository interface
type MockAssetRepository struct {
	mu     sync.Mutex
	assets map[string]domain.Asset
}

// NewMockAssetRepository creates a new mock asset repository
func NewMockAssetRepository() *MockAssetRepository {
	return &MockAssetRepository{
		assets: make(map[string]domain.Asset),
	}
}

// Save persists an asset to the mock store
func (m *MockAssetRepository) Save(ctx context.Context, asset domain.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets[asset.Filename] = asset
	return nil
}

// Get retrieves an asset by filename
func (m *MockAssetRepository) Get(ctx context.Context, filename string) (*domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	asset, ok := m.assets[filename]
	if !ok {
		return nil, fmt.Errorf("asset not found")
	}
	return &asset, nil
}

// GetByHash retrieves an asset by its content hash
func (m *MockAssetRepository) GetByHash(ctx context.Context, hash string) (*domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, asset := range m.assets {
		if asset.Hash == hash {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("asset not found")
}

// Search mock implementation
func (m *MockAssetRepository) Search(ctx context.Context, query string) ([]domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []domain.Asset
	query = strings.ToLower(query)
	for _, a := range m.assets {
		if query == "" || strings.Contains(strings.ToLower(a.Filename), query) || strings.Contains(strings.ToLower(a.Description), query) {
			results = append(results, a)
		}
	}
	return results, nil
}

// Delete removes an asset from the mock repository
func (m *MockAssetRepository) Delete(ctx context.Context, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assets, filename)
	return nil
}
