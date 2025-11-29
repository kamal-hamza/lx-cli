package mocks

import (
	"context"
	"fmt"
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

// List returns all note headers (for compatibility)
func (m *MockRepository) List(ctx context.Context) ([]domain.NoteHeader, error) {
	return m.ListHeaders(ctx)
}

// Rename renames a note from oldSlug to newTitle
func (m *MockRepository) Rename(ctx context.Context, oldSlug string, newTitle string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, ok := m.notes[oldSlug]
	if !ok {
		return fmt.Errorf("note not found: %s", oldSlug)
	}

	// Validate new title
	if err := domain.ValidateTitle(newTitle); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}

	// Generate new slug
	newSlug := domain.GenerateSlug(newTitle)

	// Check if new slug already exists
	if _, exists := m.notes[newSlug]; exists && newSlug != oldSlug {
		return fmt.Errorf("note with slug '%s' already exists", newSlug)
	}

	// Update the note
	note.Header.Title = newTitle
	note.Header.Slug = newSlug
	note.Header.Filename = domain.GenerateFilename(newSlug)

	// Move to new slug
	if newSlug != oldSlug {
		delete(m.notes, oldSlug)
		delete(m.headers, oldSlug)
	}

	m.notes[newSlug] = note
	m.headers[newSlug] = &note.Header

	return nil
}

// MockTemplateRepository is a mock implementation of the TemplateRepository interface
type MockTemplateRepository struct {
	mu        sync.RWMutex
	templates map[string]*domain.TemplateBody
}

// NewMockTemplateRepository creates a new mock template repository
func NewMockTemplateRepository() *MockTemplateRepository {
	return &MockTemplateRepository{
		templates: make(map[string]*domain.TemplateBody),
	}
}

// List returns all available templates
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

// Exists checks if a template with the given name exists
func (m *MockTemplateRepository) Exists(ctx context.Context, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.templates[name]
	return ok
}

// Get retrieves a template by name
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

// Create creates a new template
func (m *MockTemplateRepository) Create(ctx context.Context, template *domain.TemplateBody) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.templates[template.Header.Slug]; ok {
		return fmt.Errorf("template already exists: %s", template.Header.Slug)
	}

	m.templates[template.Header.Slug] = template
	return nil
}

// MockCompiler is a mock implementation of the Compiler interface for testing
type MockCompiler struct {
	mu           sync.Mutex
	calls        []string
	shouldFail   bool
	failError    error
	outputPrefix string
}

// NewMockCompiler creates a new mock compiler
func NewMockCompiler() *MockCompiler {
	return &MockCompiler{
		outputPrefix: "/fake/cache/",
	}
}

// Compile records the compile call and returns error if configured to fail
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

// GetOutputPath returns the mock output path for a slug
func (m *MockCompiler) GetOutputPath(slug string) string {
	return m.outputPrefix + slug + ".pdf"
}

// SetShouldFail configures the compiler to fail on next compile
func (m *MockCompiler) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failError = err
}

// GetCalls returns all compile calls made
func (m *MockCompiler) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// Reset clears all calls and resets state
func (m *MockCompiler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.shouldFail = false
	m.failError = nil
}

// MockPreprocessor is a mock implementation of the Preprocessor interface
type MockPreprocessor struct {
	mu         sync.Mutex
	calls      []string
	shouldFail bool
	failError  error
	mockPath   string
}

// NewMockPreprocessor creates a new mock preprocessor
func NewMockPreprocessor() *MockPreprocessor {
	return &MockPreprocessor{
		mockPath: "/fake/cache/mock_preprocessed.tex",
	}
}

// Process records the call and returns the mock path or error
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

// SetShouldFail configures the preprocessor to fail
func (m *MockPreprocessor) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failError = err
}

// SetMockPath sets the path returned by Process
func (m *MockPreprocessor) SetMockPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockPath = path
}

// GetCalls returns all process calls made
func (m *MockPreprocessor) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// Reset clears state
func (m *MockPreprocessor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.shouldFail = false
	m.failError = nil
}

// MockEditorLauncher is a mock implementation of the EditorLauncher interface
type MockEditorLauncher struct {
	mu         sync.Mutex
	openedFile string
	shouldFail bool
}

// NewMockEditorLauncher creates a new mock editor launcher
func NewMockEditorLauncher() *MockEditorLauncher {
	return &MockEditorLauncher{}
}

// Open records the file being opened and returns error if configured to fail
func (m *MockEditorLauncher) Open(ctx context.Context, filepath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.openedFile = filepath

	if m.shouldFail {
		return fmt.Errorf("failed to open editor")
	}

	return nil
}

// GetOpenedFile returns the last file that was opened
func (m *MockEditorLauncher) GetOpenedFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openedFile
}

// SetShouldFail configures the launcher to fail on next open
func (m *MockEditorLauncher) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// MockFileOpener is a mock implementation of the FileOpener interface
type MockFileOpener struct {
	mu         sync.Mutex
	openedFile string
	shouldFail bool
}

// NewMockFileOpener creates a new mock file opener
func NewMockFileOpener() *MockFileOpener {
	return &MockFileOpener{}
}

// Open records the file being opened and returns error if configured to fail
func (m *MockFileOpener) Open(ctx context.Context, filepath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.openedFile = filepath

	if m.shouldFail {
		return fmt.Errorf("failed to open file")
	}

	return nil
}

// GetOpenedFile returns the last file that was opened
func (m *MockFileOpener) GetOpenedFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openedFile
}

// SetShouldFail configures the opener to fail on next open
func (m *MockFileOpener) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}
