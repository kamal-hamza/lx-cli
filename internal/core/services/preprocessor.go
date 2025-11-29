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
func (p *Preprocessor) Process(slug string) (string, error) {
	// 1. Get Source Content
	note, err := p.repo.Get(context.Background(), slug)
	if err != nil {
		return "", err
	}
	content := note.Content

	// 2. Get All Note Headers (for lookup)
	headers, err := p.repo.ListHeaders(context.Background())
	if err != nil {
		return "", err
	}

	slugMap := make(map[string]string) // slug -> Title
	for _, h := range headers {
		slugMap[h.Slug] = h.Title
	}

	// 3. Transform References
	// Regex: \ref{something}
	refRegex := regexp.MustCompile(`\\ref\{([^}]+)\}`)

	content = refRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract slug: \ref{slug} -> slug
		submatch := refRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		targetSlug := submatch[1]

		// Check if it's a valid note
		if title, exists := slugMap[targetSlug]; exists {
			// It's a note! Replace with clickable PDF link
			// \href{./slug.pdf}{Title}
			// We use relative paths so the PDFs work anywhere
			return fmt.Sprintf(`\href{./%s.pdf}{%s}`, targetSlug, title)
		}

		// It's not a note (probably an internal \label{fig:x}), leave it alone
		return match
	})

	// 4. Ensure hyperref package exists (needed for \href)
	if !strings.Contains(content, "{hyperref}") {
		// Inject it after \documentclass
		docClassRegex := regexp.MustCompile(`(\\documentclass\[.*?\]\{.*?\})`)
		if docClassRegex.MatchString(content) {
			content = docClassRegex.ReplaceAllString(content, "$1\n\\usepackage[colorlinks=true,linkcolor=blue,urlcolor=blue,filecolor=blue]{hyperref}")
		} else {
			// Fallback: prepend
			content = "\\usepackage{hyperref}\n" + content
		}
	}

	// 5. Write to Cache
	// We write to the cache directory so we don't clutter the notes folder
	tempFilename := slug + ".tex" // Keep same name for latexmk happiness
	tempPath := filepath.Join(p.vault.CachePath, tempFilename)

	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return tempPath, nil
}
