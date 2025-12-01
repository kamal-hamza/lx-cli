package latexparser

import (
	"strings"
	"testing"
)

func TestParseLatexOutput_Success(t *testing.T) {
	output := `
This is pdfTeX, Version 3.14159265-2.6-1.40.20
Output written on test.pdf (1 page, 12345 bytes).
Transcript written on test.log.
`
	result := ParseLatexOutput(output)

	if !result.HasPDF {
		t.Error("Expected HasPDF to be true")
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

func TestParseLatexOutput_FileLineError(t *testing.T) {
	output := `
./test.tex:42: Undefined control sequence.
l.42 \invalidcommand
`
	result := ParseLatexOutput(output)

	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}

	err := result.Errors[0]
	if err.File != "./test.tex" {
		t.Errorf("Expected file './test.tex', got '%s'", err.File)
	}

	if err.Line != 42 {
		t.Errorf("Expected line 42, got %d", err.Line)
	}

	if !strings.Contains(err.Message, "Undefined control sequence") {
		t.Errorf("Expected error message to contain 'Undefined control sequence', got '%s'", err.Message)
	}
}

func TestParseLatexOutput_StandardError(t *testing.T) {
	output := `
! LaTeX Error: File 'missing.sty' not found.
`
	result := ParseLatexOutput(output)

	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(result.Errors))
	}

	err := result.Errors[0]
	if !strings.Contains(err.Message, "File 'missing.sty' not found") {
		t.Errorf("Expected error message about missing file, got '%s'", err.Message)
	}
}

func TestParseLatexOutput_Warning(t *testing.T) {
	output := `
LaTeX Warning: Citation 'key' on page 1 undefined on input line 10.
`
	result := ParseLatexOutput(output)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}

	warning := result.Warnings[0]
	if !strings.Contains(warning.Message, "Citation") {
		t.Errorf("Expected warning about citation, got '%s'", warning.Message)
	}
}

func TestParseLatexOutput_SkipBenignWarnings(t *testing.T) {
	output := `
LaTeX Warning: Reference 'ref:something' on page 1 undefined on input line 20.
LaTeX Warning: Label(s) may have changed. Rerun to get cross-references right.
`
	result := ParseLatexOutput(output)

	// Both warnings should be filtered out as benign
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings (benign filtered), got %d", len(result.Warnings))
	}
}

func TestParseLatexOutput_SkipBoxWarnings(t *testing.T) {
	output := `
Overfull \hbox (2.34pt too wide) in paragraph at lines 15--16
Underfull \vbox (badness 10000) has occurred while \output is active
`
	result := ParseLatexOutput(output)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings (box warnings filtered), got %d", len(result.Warnings))
	}
}

func TestParseLatexOutput_PackageWarning(t *testing.T) {
	output := `
Package hyperref Warning: Token not allowed in a PDF string (Unicode):
(hyperref)                removing 'math shift' on input line 50.
`
	result := ParseLatexOutput(output)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}

	warning := result.Warnings[0]
	if !strings.Contains(warning.Message, "Token not allowed") {
		t.Errorf("Expected warning about token, got '%s'", warning.Message)
	}
}

func TestParseLatexOutput_MixedErrorsAndWarnings(t *testing.T) {
	output := `
This is pdfTeX
! Undefined control sequence.
l.10 \badcommand
LaTeX Warning: Unused global option(s): [draft].
./test.tex:25: Missing $ inserted.
Output written on test.pdf (1 page, 5000 bytes).
`
	result := ParseLatexOutput(output)

	if !result.HasPDF {
		t.Error("Expected HasPDF to be true")
	}

	if len(result.Errors) < 1 {
		t.Errorf("Expected at least 1 error, got %d", len(result.Errors))
	}

	if len(result.Warnings) < 1 {
		t.Errorf("Expected at least 1 warning, got %d", len(result.Warnings))
	}
}

func TestGetSummary_Success(t *testing.T) {
	result := &ParseResult{
		HasPDF:   true,
		Errors:   []Issue{},
		Warnings: []Issue{},
	}

	summary := result.GetSummary()
	if !strings.Contains(summary, "✅") {
		t.Errorf("Expected success indicator in summary, got '%s'", summary)
	}
}

func TestGetSummary_SuccessWithWarnings(t *testing.T) {
	result := &ParseResult{
		HasPDF: true,
		Errors: []Issue{},
		Warnings: []Issue{
			{Level: LevelWarning, Message: "Test warning"},
		},
	}

	summary := result.GetSummary()
	if !strings.Contains(summary, "warning") {
		t.Errorf("Expected 'warning' in summary, got '%s'", summary)
	}
}

func TestGetSummary_Failure(t *testing.T) {
	result := &ParseResult{
		HasPDF: false,
		Errors: []Issue{
			{Level: LevelError, Message: "Test error"},
		},
		Warnings: []Issue{},
	}

	summary := result.GetSummary()
	if !strings.Contains(summary, "failed") {
		t.Errorf("Expected 'failed' in summary, got '%s'", summary)
	}
}

func TestGetSummary_PDFWithErrors(t *testing.T) {
	result := &ParseResult{
		HasPDF: true,
		Errors: []Issue{
			{Level: LevelError, Message: "Non-critical error"},
		},
		Warnings: []Issue{},
	}

	summary := result.GetSummary()
	if !strings.Contains(summary, "PDF generated") {
		t.Errorf("Expected 'PDF generated' in summary, got '%s'", summary)
	}
	if !strings.Contains(summary, "error") {
		t.Errorf("Expected 'error' in summary, got '%s'", summary)
	}
}

func TestFormatIssue_Error(t *testing.T) {
	issue := Issue{
		Level:   LevelError,
		File:    "test.tex",
		Line:    42,
		Message: "Something went wrong",
	}

	formatted := FormatIssue(issue)
	if !strings.Contains(formatted, "ERROR") {
		t.Errorf("Expected 'ERROR' in formatted output, got '%s'", formatted)
	}
	if !strings.Contains(formatted, "test.tex:42") {
		t.Errorf("Expected file:line in formatted output, got '%s'", formatted)
	}
	if !strings.Contains(formatted, "Something went wrong") {
		t.Errorf("Expected message in formatted output, got '%s'", formatted)
	}
}

func TestFormatIssue_Warning(t *testing.T) {
	issue := Issue{
		Level:   LevelWarning,
		Message: "This is a warning",
	}

	formatted := FormatIssue(issue)
	if !strings.Contains(formatted, "WARNING") {
		t.Errorf("Expected 'WARNING' in formatted output, got '%s'", formatted)
	}
	if !strings.Contains(formatted, "This is a warning") {
		t.Errorf("Expected message in formatted output, got '%s'", formatted)
	}
}

func TestIsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		hasPDF   bool
		expected bool
	}{
		{"PDF generated", true, true},
		{"No PDF", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ParseResult{HasPDF: tt.hasPDF}
			if got := result.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetCriticalIssues(t *testing.T) {
	result := &ParseResult{
		Errors: []Issue{
			{Level: LevelError, Message: "Error 1"},
			{Level: LevelError, Message: "Error 2"},
		},
		Warnings: []Issue{
			{Level: LevelWarning, Message: "Warning 1"},
		},
	}

	critical := result.GetCriticalIssues()
	if len(critical) != 2 {
		t.Errorf("Expected 2 critical issues (errors), got %d", len(critical))
	}

	for _, issue := range critical {
		if issue.Level != LevelError {
			t.Errorf("Expected only errors in critical issues, got level %v", issue.Level)
		}
	}
}
