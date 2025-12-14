package metadata

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Metadata represents the structured metadata from a note file
type Metadata struct {
	Title string
	Date  string
	Tags  []string
}

// ParseError represents a metadata parsing error
type ParseError struct {
	Line    int
	Field   string
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d: %s - %s", e.Line, e.Field, e.Message)
}

// ParseResult contains the parsing outcome with detailed error information
type ParseResult struct {
	Metadata *Metadata
	Errors   []ParseError
	Warnings []string
}

// Parser handles metadata extraction from LaTeX files
type Parser struct {
	strict bool
}

// NewParser creates a new metadata parser
func NewParser(strict bool) *Parser {
	return &Parser{
		strict: strict,
	}
}

// Parse extracts metadata from file content
func (p *Parser) Parse(content string) (*ParseResult, error) {
	result := &ParseResult{
		Metadata: &Metadata{Tags: []string{}},
		Errors:   []ParseError{},
		Warnings: []string{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	// Regex helpers
	reTitle := regexp.MustCompile(`(?i)^%\s*Title:\s*(.+)`)
	reDate := regexp.MustCompile(`(?i)^%\s*Date:\s*(.+)`)
	reTags := regexp.MustCompile(`(?i)^%\s*Tags:\s*(.+)`)

	foundTitle := false
	foundDate := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Stop scanning if we hit the document class (optimization)
		if strings.HasPrefix(line, "\\documentclass") {
			break
		}

		// Title
		if matches := reTitle.FindStringSubmatch(line); len(matches) > 1 {
			result.Metadata.Title = strings.TrimSpace(matches[1])
			foundTitle = true
		}

		// Date
		if matches := reDate.FindStringSubmatch(line); len(matches) > 1 {
			result.Metadata.Date = strings.TrimSpace(matches[1])
			foundDate = true
		}

		// Tags
		if matches := reTags.FindStringSubmatch(line); len(matches) > 1 {
			tagsStr := matches[1]
			parts := strings.Split(tagsStr, ",")
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					result.Metadata.Tags = append(result.Metadata.Tags, t)
				}
			}
		}
	}

	// Validation
	if !foundTitle {
		err := ParseError{Line: 0, Field: "Title", Message: "missing mandatory field"}
		result.Errors = append(result.Errors, err)
	}

	// In strict mode, Date is also required
	if p.strict && !foundDate {
		err := ParseError{Line: 0, Field: "Date", Message: "missing mandatory field"}
		result.Errors = append(result.Errors, err)
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("parsing failed with %d errors", len(result.Errors))
	}

	return result, nil
}

// Extract is a convenience function for non-strict parsing
func Extract(content string) (*Metadata, error) {
	parser := NewParser(false)
	result, err := parser.Parse(content)
	if err != nil {
		return nil, err
	}
	return result.Metadata, nil
}

// ExtractStrict is a convenience function for strict parsing
func ExtractStrict(content string) (*Metadata, error) {
	parser := NewParser(true)
	result, err := parser.Parse(content)
	if err != nil {
		return nil, err
	}
	return result.Metadata, nil
}

// -----------------------------------------------------------------------------
// New Functionality (Merged from utils.go)
// -----------------------------------------------------------------------------

// Format generates the standard LaTeX comment block for metadata
func Format(m *Metadata) string {
	var b strings.Builder
	b.WriteString("% ---\n")
	b.WriteString(fmt.Sprintf("%% title: %s\n", m.Title))
	b.WriteString(fmt.Sprintf("%% date: %s\n", m.Date))
	if len(m.Tags) > 0 {
		b.WriteString(fmt.Sprintf("%% tags: %s\n", strings.Join(m.Tags, ", ")))
	}
	b.WriteString("% ---\n")
	return b.String()
}

// UpdateTitle updates the title line in the content
func UpdateTitle(content, newTitle string) (string, error) {
	// Match the line starting with "% title:" or "% Title:"
	// (?m) enables multi-line mode so ^ matches start of line
	// (?i) enables case-insensitive matching
	re := regexp.MustCompile(`(?mi)(^%\s*title:\s*)(.+)`)

	if !re.MatchString(content) {
		return "", fmt.Errorf("title metadata not found in content")
	}

	// Preserve the prefix ("% title: " or "% Title: ") and replace the rest
	return re.ReplaceAllString(content, "${1}"+newTitle), nil
}
