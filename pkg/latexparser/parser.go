package latexparser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// ErrorLevel represents the severity of a LaTeX issue
type ErrorLevel int

const (
	LevelError ErrorLevel = iota
	LevelWarning
	LevelInfo
)

// Issue represents a parsed LaTeX error or warning
type Issue struct {
	Level   ErrorLevel
	File    string
	Line    int
	Message string
}

// ParseResult holds the parsed LaTeX compilation output
type ParseResult struct {
	Errors   []Issue
	Warnings []Issue
	HasPDF   bool
}

var (
	// Match error patterns like:
	// ! LaTeX Error: ...
	// ! Undefined control sequence.
	errorPattern = regexp.MustCompile(`^!\s+(.+)$`)

	// Match file-line-error format: ./file.tex:123: error message
	// This pattern is more flexible to catch various file path formats
	fileLineErrorPattern = regexp.MustCompile(`^([^:]+\.tex):(\d+):\s*(.+)$`)

	// Match warning patterns:
	// LaTeX Warning: ...
	// Package xyz Warning: ...
	warningPattern = regexp.MustCompile(`^(?:LaTeX|Package\s+\w+)\s+Warning:\s*(.+)$`)

	// Match overfull/underfull box warnings (often noise)
	boxWarningPattern = regexp.MustCompile(`^(?:Overfull|Underfull)\s+\\[hv]box`)

	// Match PDF output confirmation
	pdfOutputPattern = regexp.MustCompile(`Output written.*\.pdf`)
)

// ParseLatexOutput parses LaTeX/latexmk output and extracts meaningful issues
func ParseLatexOutput(output string) *ParseResult {
	result := &ParseResult{
		Errors:   []Issue{},
		Warnings: []Issue{},
		HasPDF:   false,
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentError string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for PDF output
		if pdfOutputPattern.MatchString(line) {
			result.HasPDF = true
		}

		// Skip box warnings (usually not critical)
		if boxWarningPattern.MatchString(line) {
			continue
		}

		// Check for file-line-error format (most reliable)
		if matches := fileLineErrorPattern.FindStringSubmatch(line); matches != nil {
			file := matches[1]
			lineNum := 0
			fmt.Sscanf(matches[2], "%d", &lineNum)
			message := matches[3]

			// Determine if it's a warning or error based on the message content
			if strings.Contains(strings.ToLower(message), "warning") {
				result.Warnings = append(result.Warnings, Issue{
					Level:   LevelWarning,
					File:    file,
					Line:    lineNum,
					Message: message,
				})
			} else {
				// File-line-error format indicates an error by default
				// These are errors reported by LaTeX with the -file-line-error flag
				result.Errors = append(result.Errors, Issue{
					Level:   LevelError,
					File:    file,
					Line:    lineNum,
					Message: message,
				})
			}
			continue
		}

		// Check for standard error pattern
		if matches := errorPattern.FindStringSubmatch(line); matches != nil {
			currentError = matches[1]
			result.Errors = append(result.Errors, Issue{
				Level:   LevelError,
				File:    "",
				Line:    0,
				Message: currentError,
			})
			continue
		}

		// Check for warning pattern
		if matches := warningPattern.FindStringSubmatch(line); matches != nil {
			message := matches[1]
			// Skip very common benign warnings
			if strings.Contains(message, "Reference") && strings.Contains(message, "undefined") {
				// Undefined references are common during multi-pass compilation
				continue
			}
			if strings.Contains(message, "Label(s) may have changed") {
				// Rerun suggestion is automatic with latexmk
				continue
			}

			result.Warnings = append(result.Warnings, Issue{
				Level:   LevelWarning,
				File:    "",
				Line:    0,
				Message: message,
			})
		}
	}

	return result
}

// FormatIssue returns a human-readable string for an issue
func FormatIssue(issue Issue) string {
	var sb strings.Builder

	switch issue.Level {
	case LevelError:
		sb.WriteString("❌ ERROR: ")
	case LevelWarning:
		sb.WriteString("⚠️  WARNING: ")
	case LevelInfo:
		sb.WriteString("ℹ️  INFO: ")
	}

	sb.WriteString(issue.Message)

	if issue.File != "" {
		sb.WriteString(fmt.Sprintf("\n   at %s", issue.File))
		if issue.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", issue.Line))
		}
	}

	return sb.String()
}

// GetSummary returns a brief summary of the parse result
func (pr *ParseResult) GetSummary() string {
	if pr.HasPDF && len(pr.Errors) == 0 {
		if len(pr.Warnings) == 0 {
			return "✅ Compilation successful"
		}
		return fmt.Sprintf("✅ PDF generated with %d warning(s)", len(pr.Warnings))
	}

	if pr.HasPDF && len(pr.Errors) > 0 {
		return fmt.Sprintf("⚠️  PDF generated but with %d error(s)", len(pr.Errors))
	}

	return fmt.Sprintf("❌ Compilation failed: %d error(s)", len(pr.Errors))
}

// GetCriticalIssues returns only errors and critical warnings
func (pr *ParseResult) GetCriticalIssues() []Issue {
	// For now, only return errors
	return pr.Errors
}

// IsSuccess returns true if compilation produced a PDF
func (pr *ParseResult) IsSuccess() bool {
	return pr.HasPDF
}
