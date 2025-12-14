package compiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/config"
	"github.com/kamal-hamza/lx-cli/pkg/latexparser"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// LatexmkCompiler implements the Compiler port using latexmk
type LatexmkCompiler struct {
	vault  *vault.Vault
	config *config.Config
}

// NewLatexmkCompiler creates a new latexmk-based compiler
func NewLatexmkCompiler(vault *vault.Vault, cfg *config.Config) *LatexmkCompiler {
	return &LatexmkCompiler{
		vault:  vault,
		config: cfg,
	}
}

// CompileResult holds compilation output and parsed issues
type CompileResult struct {
	Success    bool
	Output     string
	Parsed     *latexparser.ParseResult
	PDFPath    string
	ErrorCount int
}

// Compile compiles a note to PDF using latexmk
// The primary success criterion is: Does the PDF file exist?
// We use a multi-layered verification approach for maximum robustness
func (c *LatexmkCompiler) Compile(ctx context.Context, inputPath string, env []string) error {
	result := c.CompileWithOutput(ctx, inputPath, env)

	// CRITICAL: Check if PDF actually exists on disk
	// This is the most reliable success indicator
	if result.PDFPath != "" && fileExists(result.PDFPath) {
		// PDF exists! Compilation was successful.
		// Even if latexmk returned errors, if we have a PDF, we succeeded.
		return nil
	}

	// If we get here, no PDF was generated - this is a failure
	if result.Parsed.IsFatalError() {
		return fmt.Errorf("compilation failed: %s", result.Parsed.GetSummary())
	}

	// Fallback error message
	return fmt.Errorf("compilation failed: no PDF generated")
}

// CompileWithOutput compiles and returns detailed output for better error reporting
func (c *LatexmkCompiler) CompileWithOutput(ctx context.Context, inputPath string, env []string) *CompileResult {
	// Validate input file exists
	if !fileExists(inputPath) {
		return &CompileResult{
			Success: false,
			Output:  fmt.Sprintf("source file not found: %s", inputPath),
			Parsed: &latexparser.ParseResult{
				Errors: []latexparser.Issue{
					{
						Level:   latexparser.LevelError,
						Message: fmt.Sprintf("source file not found: %s", inputPath),
					},
				},
			},
			ErrorCount: 1,
		}
	}

	// Determine expected PDF output path
	expectedPDFPath := c.GetOutputPathFromInput(inputPath)

	// Run latexmk compilation
	output, _ := c.runLatexmk(ctx, inputPath, env)

	// Parse the output for errors and warnings
	parsed := latexparser.ParseLatexOutput(output)

	// CRITICAL: Verify PDF exists on disk (most reliable check)
	pdfExists := fileExists(expectedPDFPath)
	if pdfExists {
		parsed.HasPDF = true
		parsed.PDFPath = expectedPDFPath
	}

	// Build result
	result := &CompileResult{
		Success:    pdfExists, // Success is defined by PDF existence
		Output:     output,
		Parsed:     parsed,
		PDFPath:    expectedPDFPath,
		ErrorCount: len(parsed.Errors),
	}

	return result
}

// runLatexmk executes the latexmk command with proper configuration
func (c *LatexmkCompiler) runLatexmk(ctx context.Context, inputPath string, env []string) (string, bool) {
	// Start with flags from configuration (or defaults)
	// This allows users to switch engines (e.g. use -xelatex instead of -pdf)
	args := make([]string, len(c.config.LatexmkFlags))
	copy(args, c.config.LatexmkFlags)

	// Append mandatory flags for internal tool logic
	// -g                : force rebuild (ignore timestamps)
	// -f                : force completion even when errors occur
	// -file-line-error  : better error messages
	// -recorder         : track dependencies
	mandatoryFlags := []string{
		"-g",
		"-f",
		"-file-line-error",
		"-recorder",
		inputPath,
	}

	args = append(args, mandatoryFlags...)

	cmd := exec.CommandContext(ctx, "latexmk", args...)

	// ... [Rest of function remains the same] ...
	// Set working directory to notes path
	cmd.Dir = c.vault.NotesPath

	// Prepare environment with TEXINPUTS
	cmdEnv := os.Environ()
	texinputs := c.buildTexInputs()
	cmdEnv = append(cmdEnv, "TEXINPUTS="+texinputs)
	cmdEnv = append(cmdEnv, env...)
	cmd.Env = cmdEnv

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	cmdSuccess := (err == nil)

	return outputStr, cmdSuccess
}

// buildTexInputs constructs the TEXINPUTS environment variable
// This tells LaTeX where to find templates, assets, and other includes
func (c *LatexmkCompiler) buildTexInputs() string {
	// Format: .:templates//:assets//:notes//:
	// The // means "search recursively"
	// The trailing : means "also search default locations"

	base := c.vault.GetTexInputsEnv() // Usually returns ".:templates//:"

	// Add assets and notes directories
	parts := []string{
		base,
		c.vault.AssetsPath + "//",
		c.vault.NotesPath + "//",
	}

	return strings.Join(parts, ":")
}

// GetOutputPath returns the path to the compiled PDF for a given slug
func (c *LatexmkCompiler) GetOutputPath(slug string) string {
	// The preprocessor writes "slug.tex" to cache, so output is "slug.pdf" in cache
	return c.vault.GetCachePath(slug + ".pdf")
}

// GetOutputPathFromInput derives the PDF path from the input .tex path
func (c *LatexmkCompiler) GetOutputPathFromInput(inputPath string) string {
	if !strings.HasSuffix(inputPath, ".tex") {
		return ""
	}

	// Replace .tex extension with .pdf, keep same directory
	return strings.TrimSuffix(inputPath, ".tex") + ".pdf"
}

// Clean removes auxiliary files for a specific note
func (c *LatexmkCompiler) Clean(ctx context.Context, slug string) error {
	// Target the preprocessed/cached file for cleaning
	filename := slug + ".tex"
	targetPath := c.vault.GetCachePath(filename)

	args := []string{
		"-C", // Clean up auxiliary files
		targetPath,
	}

	cmd := exec.CommandContext(ctx, "latexmk", args...)
	cmd.Dir = c.vault.NotesPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clean failed: %w", err)
	}

	return nil
}

// IsAvailable checks if latexmk is installed and available
func IsAvailable() bool {
	_, err := exec.LookPath("latexmk")
	return err == nil
}

// fileExists checks if a file exists and is a regular file
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
