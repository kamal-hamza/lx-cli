package ports

import (
	"context"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
)

// Repository defines the port for note persistence operations
type Repository interface {
	// ListHeaders returns all note headers (lightweight operation)
	ListHeaders(ctx context.Context) ([]domain.NoteHeader, error)

	// Save persists a note to storage
	Save(ctx context.Context, note *domain.NoteBody) error

	// Get retrieves a note by slug
	Get(ctx context.Context, slug string) (*domain.NoteBody, error)

	// Exists checks if a note with the given slug exists
	Exists(ctx context.Context, slug string) bool

	// Delete removes a note by slug
	Delete(ctx context.Context, slug string) error
}

// TemplateRepository defines the port for template operations
type TemplateRepository interface {
	// List returns all available templates
	List(ctx context.Context) ([]domain.Template, error)

	// Exists checks if a template with the given name exists
	Exists(ctx context.Context, name string) bool

	// Get retrieves a template by name
	Get(ctx context.Context, name string) (*domain.Template, error)

	// Create creates a new template
	Create(ctx context.Context, template *domain.TemplateBody) error
}

// Preprocessor defines the port for note preprocessing operations
type Preprocessor interface {
	// Process creates a temporary compilable version of the note with resolved links
	// Returns the absolute path to the preprocessed file in the cache
	Process(slug string) (string, error)
}

// Compiler defines the port for LaTeX compilation operations
type Compiler interface {
	// Compile compiles a specific source file to PDF
	// inputPath: absolute path to the .tex file (usually in cache)
	// env: additional environment variables (e.g., TEXINPUTS)
	Compile(ctx context.Context, inputPath string, env []string) error

	// GetOutputPath returns the path to the compiled PDF
	GetOutputPath(slug string) string
}

// EditorLauncher defines the port for launching external editors
type EditorLauncher interface {
	// Open opens a file in the user's preferred editor
	Open(ctx context.Context, filepath string) error
}

// FileOpener defines the port for opening files with default applications
type FileOpener interface {
	// Open opens a file with the system's default application
	Open(ctx context.Context, filepath string) error
}

type AssetRepository interface {
	// Save adds or updates an asset record
	Save(ctx context.Context, asset domain.Asset) error

	// Get retrieves asset metadata by filename
	Get(ctx context.Context, filename string) (*domain.Asset, error)

	// GetByHash retrieves an asset by its content hash (NEW)
	GetByHash(ctx context.Context, hash string) (*domain.Asset, error)

	// Search finds assets matching a query
	Search(ctx context.Context, query string) ([]domain.Asset, error)

	// Delete removes an asset from the registry
	Delete(ctx context.Context, filename string) error
}
