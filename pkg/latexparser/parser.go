package latexparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Issue represents a compilation issue (error or warning)
type Issue struct {
	Level   IssueLevel
	File    string
	Line    int
	Message string
}

type IssueLevel int

const (
	LevelError IssueLevel = iota
	LevelWarning
	LevelInfo
)

func (l IssueLevel) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarning:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// ParseResult holds the parsing results
type ParseResult struct {
	Errors      []Issue
	Warnings    []Issue
	HasPDF      bool
	PDFPath     string
	CompletedOK bool // latexmk completed without fatal errors
}

var (
	// LaTeX error patterns - these indicate real problems
	errorPattern = regexp.MustCompile(`^!\s+(.+)$`)

	// File-line-error format: ./file.tex:123: error message
	fileLineErrorPattern = regexp.MustCompile(`^([^:]+\.tex):(\d+):\s*(.+)$`)

	// Warning patterns
	warningPattern = regexp.MustCompile(`^(?:LaTeX|Package\s+\w+)\s+Warning:\s*(.+)$`)

	// Box warnings (usually harmless overfull/underfull boxes)
	boxWarningPattern = regexp.MustCompile(`^(?:Overfull|Underfull)\s+\\[hv]box`)

	// PDF output confirmation - multiple patterns for robustness
	pdfOutputPatterns = []*regexp.Regexp{
		regexp.MustCompile(`Output written.*\.pdf`),
		regexp.MustCompile(`PDF file created.*\.pdf`),
		regexp.MustCompile(`^\s*Output\s+written\s+on\s+(.+\.pdf)`),
	}

	// Latexmk success indicators
	latexmkSuccessPattern = regexp.MustCompile(`Latexmk: All targets.*up-to-date`)

	// Latexmk error collection
	latexmkErrorPattern = regexp.MustCompile(`Collected error summary`)

	// Fatal LaTeX errors that prevent compilation
	fatalErrorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^!\s+Emergency stop`),
		regexp.MustCompile(`^!\s+Fatal error`),
		regexp.MustCompile(`File .* not found`),
		regexp.MustCompile(`Undefined control sequence`),
	}
)

// ParseLatexOutput parses LaTeX/latexmk output comprehensively
func ParseLatexOutput(output string) *ParseResult {
	result := &ParseResult{
		Errors:      []Issue{},
		Warnings:    []Issue{},
		HasPDF:      false,
		CompletedOK: false,
	}

	// First, join wrapped lines for better pattern matching
	unwrappedOutput := unwrapLatexOutput(output)

	scanner := bufio.NewScanner(strings.NewReader(unwrappedOutput))
	var currentError string
	inErrorContext := false
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Skip empty lines early
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for PDF output (multiple patterns for robustness)
		for _, pattern := range pdfOutputPatterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				result.HasPDF = true
				if len(matches) > 1 {
					result.PDFPath = strings.TrimSpace(matches[1])
				}
				break
			}
		}

		// Check for latexmk success
		if latexmkSuccessPattern.MatchString(line) {
			result.CompletedOK = true
		}

		// Skip box warnings (usually not critical)
		if boxWarningPattern.MatchString(line) {
			continue
		}

		// Parse file:line:error format FIRST (before other patterns)
		// This ensures we catch file:line:message before generic patterns like "Undefined control sequence"
		if matches := fileLineErrorPattern.FindStringSubmatch(line); matches != nil {
			if len(matches) >= 4 {
				file := matches[1]
				lineNumStr := matches[2]
				message := matches[3]

				// Parse the line number from the tex file
				var texLine int
				fmt.Sscanf(lineNumStr, "%d", &texLine)

				// Determine if it's actually an error or just a warning
				if strings.Contains(strings.ToLower(message), "warning") {
					result.Warnings = append(result.Warnings, Issue{
						Level:   LevelWarning,
						File:    file,
						Message: message,
						Line:    texLine,
					})
				} else {
					result.Errors = append(result.Errors, Issue{
						Level:   LevelError,
						File:    file,
						Message: message,
						Line:    texLine,
					})
				}
			}
			// IMPORTANT: Continue to next line to avoid duplicate processing
			continue
		}

		// Parse standard LaTeX errors (! format)
		if matches := errorPattern.FindStringSubmatch(line); matches != nil {
			currentError = matches[1]
			inErrorContext = true
			continue
		}

		// If we're in error context, collect the next line as it usually has details
		if inErrorContext && currentError != "" {
			result.Errors = append(result.Errors, Issue{
				Level:   LevelError,
				Message: currentError,
				Line:    lineNum,
			})
			currentError = ""
			inErrorContext = false
			continue
		}

		// Check for fatal errors (but these are now less likely to match since file:line was checked first)
		for _, pattern := range fatalErrorPatterns {
			if pattern.MatchString(line) {
				result.Errors = append(result.Errors, Issue{
					Level:   LevelError,
					Message: strings.TrimSpace(line),
					Line:    lineNum,
				})
				inErrorContext = true
				break
			}
		}

		// Parse standard warnings
		if matches := warningPattern.FindStringSubmatch(line); matches != nil {
			result.Warnings = append(result.Warnings, Issue{
				Level:   LevelWarning,
				Message: matches[1],
				Line:    lineNum,
			})
			continue
		}
	}

	// If we have a pending error that wasn't followed by details, add it now
	if currentError != "" {
		result.Errors = append(result.Errors, Issue{
			Level:   LevelError,
			Message: currentError,
			Line:    lineNum,
		})
	}

	return result
}

// unwrapLatexOutput attempts to join wrapped lines in LaTeX output
// LaTeX output often wraps long lines at 79 characters, breaking pattern matching
// We use a conservative approach - only join lines that are clearly continuations
func unwrapLatexOutput(output string) string {
	lines := strings.Split(output, "\n")
	var unwrapped []string
	var buffer string

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")

		// Only join if the previous line looks incomplete (ends mid-path or mid-word)
		// and the current line doesn't look like a new statement
		shouldJoin := false
		if buffer != "" {
			// Check if buffer looks like it was cut off (ends with path separator or incomplete path)
			if strings.HasSuffix(buffer, "/") ||
				(strings.Contains(buffer, "Output written") && !strings.HasSuffix(buffer, ".pdf")) {
				shouldJoin = true
			}
		}

		if shouldJoin && !looksLikeNewStatement(line) {
			buffer += trimmed
		} else {
			if buffer != "" {
				unwrapped = append(unwrapped, buffer)
			}
			buffer = trimmed
		}
	}

	if buffer != "" {
		unwrapped = append(unwrapped, buffer)
	}

	return strings.Join(unwrapped, "\n")
}

// looksLikeNewStatement checks if a line starts a new statement (not a continuation)
func looksLikeNewStatement(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return true
	}

	// Common line start patterns
	patterns := []string{
		"!",         // Error
		"LaTeX",     // LaTeX message
		"Package",   // Package message
		"Output",    // Output message
		"(",         // File opening
		")",         // File closing
		"<",         // File reference
		">",         // File reference
		"Latexmk:",  // Latexmk message
		"Running",   // Command execution
		"Overfull",  // Box warning
		"Underfull", // Box warning
		"Chapter",   // Structure
		"Section",   // Structure
		"[",         // Page number
		"*",         // Some messages
		"This is",   // Program identification
		"File:",     // File reference
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(trimmed, pattern) {
			return true
		}
	}

	// Check if it looks like a file path
	if strings.Contains(trimmed, "/") && strings.Contains(trimmed, ".tex") {
		return true
	}

	// Check if it matches common message patterns
	if strings.Contains(trimmed, ":") {
		return true
	}

	return false
}

// IsSuccess returns true if compilation was successful (PDF exists)
func (pr *ParseResult) IsSuccess() bool {
	return pr.HasPDF
}

// IsFatalError returns true if there are critical errors that prevented PDF generation
func (pr *ParseResult) IsFatalError() bool {
	return !pr.HasPDF && len(pr.Errors) > 0
}

// GetSummary returns a human-readable summary of the compilation
func (pr *ParseResult) GetSummary() string {
	if pr.HasPDF && len(pr.Errors) == 0 && len(pr.Warnings) == 0 {
		return "✅ Compilation successful"
	}

	if pr.HasPDF && len(pr.Errors) == 0 && len(pr.Warnings) > 0 {
		return fmt.Sprintf("✅ PDF generated with %d warning(s)", len(pr.Warnings))
	}

	if pr.HasPDF && len(pr.Errors) > 0 {
		return fmt.Sprintf("⚠️  PDF generated but LaTeX reported %d issue(s)", len(pr.Errors))
	}

	if !pr.HasPDF && len(pr.Errors) > 0 {
		return fmt.Sprintf("❌ Compilation failed with %d error(s)", len(pr.Errors))
	}

	if !pr.HasPDF {
		return "❌ Compilation failed (no PDF generated)"
	}

	return "⚠️  Unknown compilation status"
}

// FormatIssues returns a formatted string of all issues
func (pr *ParseResult) FormatIssues() string {
	if len(pr.Errors) == 0 && len(pr.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder

	if len(pr.Errors) > 0 {
		sb.WriteString("\nErrors:\n")
		for i, err := range pr.Errors {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("  ... and %d more errors\n", len(pr.Errors)-10))
				break
			}
			if err.File != "" {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", err.File, err.Message))
			} else {
				sb.WriteString(fmt.Sprintf("  • %s\n", err.Message))
			}
		}
	}

	if len(pr.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for i, warn := range pr.Warnings {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("  ... and %d more warnings\n", len(pr.Warnings)-5))
				break
			}
			if warn.File != "" {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", warn.File, warn.Message))
			} else {
				sb.WriteString(fmt.Sprintf("  • %s\n", warn.Message))
			}
		}
	}

	return sb.String()
}

// VerifyPDFExists checks if the PDF file actually exists on disk
// This is the most reliable way to determine compilation success
func VerifyPDFExists(pdfPath string) bool {
	if pdfPath == "" {
		return false
	}

	info, err := os.Stat(pdfPath)
	if err != nil {
		return false
	}

	// Check if it's a regular file and has non-zero size
	return info.Mode().IsRegular() && info.Size() > 0
}

// ExtractPDFPath attempts to extract the PDF path from the expected cache location
func ExtractPDFPath(inputPath string) string {
	// Input is usually /path/to/cache/slug.tex
	// Output should be /path/to/cache/slug.pdf
	if !strings.HasSuffix(inputPath, ".tex") {
		return ""
	}

	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	slug := strings.TrimSuffix(base, ".tex")

	return filepath.Join(dir, slug+".pdf")
}
