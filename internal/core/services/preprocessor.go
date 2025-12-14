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

// resolveReferences converts \lxnote{slug} and \ref{slug} (deprecated) to \href{./slug.pdf}{Title}
func (p *Preprocessor) resolveReferences(content string, slugMap map[string]string) string {
	// Primary: \lxnote[optional text]{slug} or \lxnote{slug}
	// Regex explanation:
	//   \\lxnote       : Matches literal "\lxnote"
	//   (?:\[(.*?)\])? : Non-capturing group for optional [text].
	//                    (.*?) captures the content lazily into submatch 1.
	//   \{([^}]+)\}    : Matches {slug}. Captures slug into submatch 2.
	lxnoteRegex := regexp.MustCompile(`\\lxnote(?:\[(.*?)\])?\{([^}]+)\}`)

	content = lxnoteRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := lxnoteRegex.FindStringSubmatch(match)
		// submatches[0] = full match
		// submatches[1] = optional text (might be empty)
		// submatches[2] = slug

		if len(submatches) < 3 {
			return match
		}

		customText := strings.TrimSpace(submatches[1])
		targetSlug := strings.TrimSpace(submatches[2])

		// 1. Resolve Target Title
		targetTitle, exists := slugMap[targetSlug]
		if !exists {
			return fmt.Sprintf(`\textbf{[BROKEN LINK: %s]}`, targetSlug)
		}

		// 2. Determine Display Text
		// If user provided [custom text], use it. Otherwise, use the note's Title.
		displayText := targetTitle
		if customText != "" {
			displayText = customText
		}

		// 3. Generate Hyperref
		// We use relative paths so it works in the PDF viewer
		return fmt.Sprintf(`\href{./%s.pdf}{%s}`, targetSlug, displayText)
	})

	// Legacy: \ref{slug} - deprecated, only converts if it matches a note slug
	// This provides backward compatibility but will be removed in a future version
	refRegex := regexp.MustCompile(`\\ref\{([^}]+)\}`)
	content = refRegex.ReplaceAllStringFunc(content, func(match string) string {
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

	return content
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
	// Check if hyperref is already loaded in the note itself
	if strings.Contains(content, "{hyperref}") {
		return content
	}

	// Check if any loaded templates contain hyperref
	// Extract all \usepackage{...} statements
	usepackageRegex := regexp.MustCompile(`\\usepackage\{([^}]+)\}`)
	matches := usepackageRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			packageName := match[1]
			// Check if this is a .sty file in the templates directory
			templatePath := filepath.Join(p.vault.TemplatesPath, packageName+".sty")
			if templateContent, err := os.ReadFile(templatePath); err == nil {
				// If the template file contains hyperref, don't inject it
				if strings.Contains(string(templateContent), "hyperref") {
					return content
				}
			}
		}
	}

	// Inject after \documentclass (with or without optional parameters)
	docClassRegex := regexp.MustCompile(`(\\documentclass(?:\[.*?\])?\{.*?\})`)

	if docClassRegex.MatchString(content) {
		// Add with standard options for nice links
		return docClassRegex.ReplaceAllString(content, "$1\n\\usepackage[colorlinks=true,linkcolor=blue,urlcolor=blue,filecolor=blue]{hyperref}")
	}

	// Fallback: prepend to file
	return "\\usepackage[colorlinks=true,linkcolor=blue,urlcolor=blue,filecolor=blue]{hyperref}\n" + content
}
