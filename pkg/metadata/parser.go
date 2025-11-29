package metadata

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Metadata represents the structured metadata from a note file
type Metadata struct {
	Title string
	Date  string
	Tags  []string
}

// ParseResult contains the parsing outcome with detailed error information
type ParseResult struct {
	Metadata *Metadata
	Errors   []ParseError
	Warnings []string
}

// ParseError represents a metadata parsing error
type ParseError struct {
	Line    int
	Field   string
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d (%s): %s", e.Line, e.Field, e.Message)
}

// Parser handles metadata extraction from LaTeX files
type Parser struct {
	strict bool // If true, fail on any error; if false, try to recover
}

// NewParser creates a new metadata parser
func NewParser(strict bool) *Parser {
	return &Parser{strict: strict}
}

// Parse extracts metadata from file content
// Returns metadata (possibly partial if not strict) and any errors/warnings
func (p *Parser) Parse(content string) (*ParseResult, error) {
	result := &ParseResult{
		Metadata: &Metadata{
			Tags: []string{},
		},
		Errors:   []ParseError{},
		Warnings: []string{},
	}

	// Find metadata block
	metadataBlock, blockStart, found := p.extractMetadataBlock(content)
	if !found {
		err := ParseError{
			Line:    0,
			Field:   "metadata",
			Message: "no metadata block found",
		}
		result.Errors = append(result.Errors, err)
		if p.strict {
			return result, fmt.Errorf("no metadata block found")
		}
		return result, nil
	}

	// Parse metadata fields
	scanner := bufio.NewScanner(strings.NewReader(metadataBlock))
	lineNum := blockStart

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and metadata header
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%% Metadata") || strings.HasPrefix(trimmed, "% Metadata") {
			continue
		}

		// Parse metadata line
		if err := p.parseMetadataLine(line, lineNum, result); err != nil {
			if p.strict {
				return result, err
			}
		}
	}

	// Validate required fields
	if err := p.validateMetadata(result.Metadata); err != nil {
		result.Errors = append(result.Errors, ParseError{
			Line:    0,
			Field:   "validation",
			Message: err.Error(),
		})
		if p.strict {
			return result, err
		}
	}

	// Normalize metadata
	p.normalizeMetadata(result.Metadata)

	return result, nil
}

// extractMetadataBlock finds the metadata comment block at the start of the file
// Returns the block content, starting line number, and whether it was found
func (p *Parser) extractMetadataBlock(content string) (string, int, bool) {
	lines := strings.Split(content, "\n")
	var metadataLines []string
	inMetadata := false
	startLine := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of metadata block - check if line contains "Metadata" after removing % chars
		if !inMetadata {
			// Remove all leading % characters and whitespace
			stripped := trimmed
			for strings.HasPrefix(stripped, "%") {
				stripped = strings.TrimPrefix(stripped, "%")
				stripped = strings.TrimSpace(stripped)
			}
			if strings.EqualFold(stripped, "Metadata") || strings.HasPrefix(strings.ToLower(stripped), "metadata") {
				inMetadata = true
				startLine = i
				metadataLines = append(metadataLines, line)
				continue
			}
		}

		// Inside metadata block
		if inMetadata {
			// End of metadata block (non-comment line or empty line after fields)
			if !strings.HasPrefix(trimmed, "%") {
				break
			}
			metadataLines = append(metadataLines, line)
		}
	}

	if len(metadataLines) == 0 {
		return "", 0, false
	}

	return strings.Join(metadataLines, "\n"), startLine, true
}

// parseMetadataLine parses a single metadata line
func (p *Parser) parseMetadataLine(line string, lineNum int, result *ParseResult) error {
	// Match format: % field: value or %% field: value
	// Need to handle lines with multiple % signs and various whitespace
	trimmed := strings.TrimSpace(line)

	// Remove leading % characters
	for strings.HasPrefix(trimmed, "%") {
		trimmed = strings.TrimPrefix(trimmed, "%")
		trimmed = strings.TrimSpace(trimmed)
	}

	// Now parse field: value format
	re := regexp.MustCompile(`^(\w+):\s*(.*)$`)
	matches := re.FindStringSubmatch(trimmed)

	if matches == nil {
		// Line doesn't match expected format
		err := ParseError{
			Line:    lineNum,
			Field:   "format",
			Message: fmt.Sprintf("invalid metadata line format: %s", line),
		}
		result.Warnings = append(result.Warnings, err.Error())
		return nil // Don't fail on format issues, just warn
	}

	field := strings.ToLower(strings.TrimSpace(matches[1]))
	value := strings.TrimSpace(matches[2])

	switch field {
	case "title":
		if result.Metadata.Title != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: duplicate title field, using first occurrence", lineNum))
			return nil
		}
		if value == "" {
			err := ParseError{
				Line:    lineNum,
				Field:   "title",
				Message: "title cannot be empty",
			}
			result.Errors = append(result.Errors, err)
			return fmt.Errorf("title cannot be empty")
		}
		result.Metadata.Title = value

	case "date":
		if result.Metadata.Date != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: duplicate date field, using first occurrence", lineNum))
			return nil
		}
		if err := p.validateDate(value); err != nil {
			parseErr := ParseError{
				Line:    lineNum,
				Field:   "date",
				Message: err.Error(),
			}
			result.Errors = append(result.Errors, parseErr)
			// Still store the date even if invalid format
			result.Metadata.Date = value
			if p.strict {
				return err
			}
		} else {
			result.Metadata.Date = value
		}

	case "tags":
		if len(result.Metadata.Tags) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: duplicate tags field, merging values", lineNum))
		}
		// Parse comma-separated tags
		if value != "" {
			tags := strings.Split(value, ",")
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					result.Metadata.Tags = append(result.Metadata.Tags, trimmed)
				}
			}
		}

	default:
		result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: unknown metadata field '%s', ignoring", lineNum, field))
	}

	return nil
}

// validateDate checks if a date string is in valid format (YYYY-MM-DD)
func (p *Parser) validateDate(date string) error {
	if date == "" {
		return nil // Empty date is allowed
	}

	// Try parsing as YYYY-MM-DD
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %s", date)
	}

	return nil
}

// validateMetadata checks required fields are present
func (p *Parser) validateMetadata(m *Metadata) error {
	if m.Title == "" {
		return fmt.Errorf("required field 'title' is missing")
	}

	return nil
}

// normalizeMetadata cleans up and normalizes metadata values
func (p *Parser) normalizeMetadata(m *Metadata) {
	// Trim and clean title
	m.Title = strings.TrimSpace(m.Title)

	// Normalize date format
	if m.Date != "" {
		m.Date = strings.TrimSpace(m.Date)
	}

	// Remove duplicate tags and normalize case
	tagSet := make(map[string]bool)
	var uniqueTags []string
	for _, tag := range m.Tags {
		normalized := strings.TrimSpace(strings.ToLower(tag))
		if normalized != "" && !tagSet[normalized] {
			tagSet[normalized] = true
			uniqueTags = append(uniqueTags, tag) // Keep original case
		}
	}
	m.Tags = uniqueTags
}

// Format generates a standardized metadata block
func Format(m *Metadata) string {
	var builder strings.Builder

	builder.WriteString("%% Metadata\n")
	builder.WriteString(fmt.Sprintf("%%%% title: %s\n", m.Title))

	if m.Date != "" {
		builder.WriteString(fmt.Sprintf("%%%% date: %s\n", m.Date))
	} else {
		builder.WriteString(fmt.Sprintf("%%%% date: %s\n", time.Now().Format("2006-01-02")))
	}

	if len(m.Tags) > 0 {
		builder.WriteString(fmt.Sprintf("%%%% tags: %s\n", strings.Join(m.Tags, ", ")))
	} else {
		builder.WriteString("%% tags: \n")
	}

	return builder.String()
}

// Update replaces or adds metadata to content
// If metadata exists, it's replaced; otherwise it's prepended
func Update(content string, m *Metadata) string {
	parser := NewParser(false)
	_, blockStart, found := parser.extractMetadataBlock(content)

	newMetadata := Format(m)

	if !found {
		// No existing metadata, prepend it
		return newMetadata + "\n" + content
	}

	// Replace existing metadata
	lines := strings.Split(content, "\n")

	// Find end of metadata block
	blockEnd := blockStart
	inBlock := false
	for i, line := range lines {
		if i < blockStart {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "%% Metadata") || strings.HasPrefix(trimmed, "% Metadata") {
			inBlock = true
			continue
		}

		if inBlock && !strings.HasPrefix(trimmed, "%") {
			blockEnd = i
			break
		}

		if inBlock && strings.HasPrefix(trimmed, "%") {
			blockEnd = i + 1
		}
	}

	// Reconstruct content
	var result strings.Builder

	// Lines before metadata
	for i := 0; i < blockStart; i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	// New metadata
	result.WriteString(newMetadata)

	// Lines after metadata
	for i := blockEnd; i < len(lines); i++ {
		if i == blockEnd && strings.TrimSpace(lines[i]) == "" {
			// Skip empty line after metadata if present
			continue
		}
		result.WriteString(lines[i])
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
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
