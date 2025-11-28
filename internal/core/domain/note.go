package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// NoteHeader represents the lightweight metadata of a note
// Used for listing operations to avoid loading full content
type NoteHeader struct {
	Title    string   `yaml:"title"`
	Date     string   `yaml:"date"`
	Tags     []string `yaml:"tags"`
	Slug     string   `yaml:"-"` // e.g., "graph-theory"
	Filename string   `yaml:"-"` // e.g., "20251128-graph-theory.tex"
}

// NoteBody represents the full note with content
// Used for build and open operations
type NoteBody struct {
	Header  NoteHeader
	Content string // The full LaTeX source
}

// Template represents a .sty file in the templates directory
type Template struct {
	Name string // "homework" (from homework.sty)
	Path string // Absolute path
}

// GenerateSlug creates a URL-friendly slug from a title
// Converts "Graph Theory Notes" -> "graph-theory-notes"
func GenerateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Collapse multiple hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// GenerateFilename creates a filename from date and slug
// Format: YYYYMMDD-slug.tex
func GenerateFilename(slug string) string {
	date := time.Now().Format("20060102")
	return fmt.Sprintf("%s-%s.tex", date, slug)
}

// ParseFilename extracts slug from filename
// "20251128-graph-theory.tex" -> "graph-theory"
func ParseFilename(filename string) string {
	// Remove .tex extension
	name := strings.TrimSuffix(filename, ".tex")

	// Find first hyphen (after date)
	parts := strings.SplitN(name, "-", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return name
}

// ValidateTitle checks if a title is valid
func ValidateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title cannot be empty")
	}

	if len(title) > 200 {
		return fmt.Errorf("title too long (max 200 characters)")
	}

	return nil
}

// ValidateTemplate checks if a template name is valid
func ValidateTemplate(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	// Check for valid filename characters
	reg := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !reg.MatchString(name) {
		return fmt.Errorf("template name contains invalid characters")
	}

	return nil
}

// NewNoteHeader creates a new note header
func NewNoteHeader(title string, tags []string) (*NoteHeader, error) {
	if err := ValidateTitle(title); err != nil {
		return nil, err
	}

	slug := GenerateSlug(title)
	filename := GenerateFilename(slug)
	date := time.Now().Format("2006-01-02")

	if tags == nil {
		tags = []string{}
	}

	return &NoteHeader{
		Title:    title,
		Date:     date,
		Tags:     tags,
		Slug:     slug,
		Filename: filename,
	}, nil
}

// NewNoteBody creates a new note body with header and content
func NewNoteBody(header *NoteHeader, content string) *NoteBody {
	return &NoteBody{
		Header:  *header,
		Content: content,
	}
}

// HasTag checks if the note has a specific tag
func (h *NoteHeader) HasTag(tag string) bool {
	for _, t := range h.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// GetDisplayDate returns a human-readable date
func (h *NoteHeader) GetDisplayDate() string {
	t, err := time.Parse("2006-01-02", h.Date)
	if err != nil {
		return h.Date
	}
	return t.Format("Jan 02, 2006")
}

// GetTagsString returns tags as a comma-separated string
func (h *NoteHeader) GetTagsString() string {
	if len(h.Tags) == 0 {
		return "-"
	}
	return strings.Join(h.Tags, ", ")
}
