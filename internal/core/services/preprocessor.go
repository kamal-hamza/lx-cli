package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/ports"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

type Preprocessor struct {
	repo  ports.Repository
	vault *vault.Vault
}

func NewPreprocessor(repo ports.Repository, v *vault.Vault) *Preprocessor {
	return &Preprocessor{
		repo:  repo,
		vault: v,
	}
}

// Process creates a temporary compilable version of the note with resolved links
// Returns the absolute path to the preprocessed file in the cache
func (p *Preprocessor) Process(slug string) (string, error) {
	// 1. Get Source Content
	note, err := p.repo.Get(context.Background(), slug)
	if err != nil {
		return "", err
	}
	content := note.Content

	// 2. Get All Note Headers (for cross-reference lookup)
	headers, err := p.repo.ListHeaders(context.Background())
	if err != nil {
		return "", err
	}

	slugMap := make(map[string]string) // slug -> Title
	for _, h := range headers {
		slugMap[h.Slug] = h.Title
	}

	// 3. Process Content
	content = p.resolveReferences(content, slugMap)
	content = p.resolveInputs(content)
	content = p.resolveGraphics(content)
	content = p.ensureHyperref(content)

	// 4. Write to Cache
	// We write to the cache directory so we don't clutter the notes folder
	tempFilename := slug + ".tex"
	tempPath := filepath.Join(p.vault.CachePath, tempFilename)

	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write preprocessed file: %w", err)
	}

	return tempPath, nil
}

// resolveReferences converts \ref{slug} to \href{./slug.pdf}{Title}
func (p *Preprocessor) resolveReferences(content string, slugMap map[string]string) string {
	refRegex := regexp.MustCompile(`\\ref\{([^}]+)\}`)

	return refRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatch := refRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		targetSlug := submatch[1]

		// Check if the reference target matches a known note slug
		if title, exists := slugMap[targetSlug]; exists {
			// It's a note! Replace with clickable PDF link
			// We use relative paths ./slug.pdf so the links work in the PDF viewer
			return fmt.Sprintf(`\href{./%s.pdf}{%s}`, targetSlug, title)
		}

		// It's not a note (likely a standard internal label like \label{fig:x}), leave it alone
		return match
	})
}

// resolveInputs converts relative \input{...} paths to absolute paths
func (p *Preprocessor) resolveInputs(content string) string {
	// Matches \input{filename} or \include{filename}
	inputRegex := regexp.MustCompile(`\\(input|include)\{([^}]+)\}`)

	return inputRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := inputRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		command := parts[1]
		path := parts[2]

		// If path is already absolute, leave it
		if filepath.IsAbs(path) {
			return match
		}

		// Resolve relative path against the Notes directory
		absPath := filepath.Join(p.vault.NotesPath, path)
		// Ensure we use forward slashes for LaTeX compatibility
		absPath = filepath.ToSlash(absPath)

		return fmt.Sprintf(`\%s{%s}`, command, absPath)
	})
}

// resolveGraphics fixes relative paths in \includegraphics
func (p *Preprocessor) resolveGraphics(content string) string {
	// Matches \includegraphics[options]{path}
	// The optional argument [.*?] is lazy
	imgRegex := regexp.MustCompile(`\\includegraphics(\[.*?\])?\{([^}]+)\}`)

	return imgRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := imgRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		opts := parts[1] // e.g. [width=0.5\textwidth] or empty string
		path := parts[2]

		// If path is absolute, leave it
		if filepath.IsAbs(path) {
			return match
		}

		// If path is relative (starts with . or ..), resolve it against NotesPath
		// Standard assets without ./ are usually handled by TEXINPUTS, but resolving them
		// to the assets dir here makes it robust.
		if strings.HasPrefix(path, ".") {
			absPath := filepath.Join(p.vault.NotesPath, path)
			absPath = filepath.ToSlash(absPath)
			return fmt.Sprintf(`\includegraphics%s{%s}`, opts, absPath)
		}

		// If just a filename, assume it might be in assets/
		// (Optional: You could check if file exists in assets/ here)

		return match
	})
}

// ensureHyperref injects the hyperref package if missing
func (p *Preprocessor) ensureHyperref(content string) string {
	if strings.Contains(content, "{hyperref}") {
		return content
	}

	// Inject after \documentclass
	docClassRegex := regexp.MustCompile(`(\\documentclass\[.*?\]\{.*?\})`)

	if docClassRegex.MatchString(content) {
		// Add with standard options for nice links
		return docClassRegex.ReplaceAllString(content, "$1\n\\usepackage[colorlinks=true,linkcolor=blue,urlcolor=blue,filecolor=blue]{hyperref}")
	}

	// Fallback: prepend to file
	return "\\usepackage[colorlinks=true,linkcolor=blue,urlcolor=blue,filecolor=blue]{hyperref}\n" + content
}
