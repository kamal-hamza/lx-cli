package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// NoteHeader represents the lightweight metadata of a note
type NoteHeader struct {
	Title    string   `yaml:"title"`
	Date     string   `yaml:"date"`
	Tags     []string `yaml:"tags"`
	Slug     string   `yaml:"-"`
	Filename string   `yaml:"-"`
}

// NoteBody represents the full note with content
type NoteBody struct {
	Header  NoteHeader
	Content string
}

// Template represents a .sty file in the templates directory
type Template struct {
	Name string
	Path string
}

// GenerateSlug creates a URL-friendly slug from a title
func GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")
	return slug
}

// GenerateFilename creates a filename from date and slug using the provided format
// Format example: "20060102" or "2006-01-02"
func GenerateFilename(slug string, dateFormat string) string {
	if dateFormat == "" {
		// No date prefix
		return fmt.Sprintf("%s.tex", slug)
	}
	date := time.Now().Format(dateFormat)
	return fmt.Sprintf("%s-%s.tex", date, slug)
}

// ParseFilename extracts slug from filename, handling various date formats
// "20251128-graph-theory.tex" -> "graph-theory"
// "2025-11-28-graph-theory.tex" -> "graph-theory"
// "graph-theory.tex" -> "graph-theory"
func ParseFilename(filename string) string {
	// Remove .tex extension
	name := strings.TrimSuffix(filename, ".tex")

	// Regex to match common date prefixes followed by a hyphen
	// Matches:
	// - 8 digits (YYYYMMDD) e.g., 20251128-
	// - YYYY-MM-DD e.g., 2025-11-28-
	dateRegex := regexp.MustCompile(`^(\d{8}|\d{4}-\d{2}-\d{2})-(.+)$`)

	matches := dateRegex.FindStringSubmatch(name)
	if len(matches) == 3 {
		// Group 2 is the slug
		return matches[2]
	}

	// Fallback for other formats or no date: just return name
	// If name contains hyphens but doesn't match date regex, assume it's just the slug
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
	reg := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !reg.MatchString(name) {
		return fmt.Errorf("template name contains invalid characters")
	}
	return nil
}

// NewNoteHeader creates a new note header with the specified date format
func NewNoteHeader(title string, tags []string, dateFormat string) (*NoteHeader, error) {
	if err := ValidateTitle(title); err != nil {
		return nil, err
	}

	slug := GenerateSlug(title)
	filename := GenerateFilename(slug, dateFormat)
	// Internal metadata date is always YYYY-MM-DD for consistency
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

// GetDisplayDate returns a human-readable date using the provided format
// If format is empty, defaults to "Jan 02, 2006"
func (h *NoteHeader) GetDisplayDate(format string) string {
	t, err := time.Parse("2006-01-02", h.Date)
	if err != nil {
		return h.Date
	}
	if format == "" {
		format = "Jan 02, 2006"
	}
	return t.Format(format)
}

// GetTagsString returns tags as a comma-separated string
func (h *NoteHeader) GetTagsString() string {
	if len(h.Tags) == 0 {
		return "-"
	}
	return strings.Join(h.Tags, ", ")
}
