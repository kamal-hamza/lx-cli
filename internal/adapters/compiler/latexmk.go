package compiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/kamal-hamza/lx-cli/pkg/latexparser"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// LatexmkCompiler implements the Compiler port using latexmk
type LatexmkCompiler struct {
	vault *vault.Vault
}

// NewLatexmkCompiler creates a new latexmk-based compiler
func NewLatexmkCompiler(vault *vault.Vault) *LatexmkCompiler {
	return &LatexmkCompiler{
		vault: vault,
	}
}

// CompileResult holds compilation output and parsed issues
type CompileResult struct {
	Success bool
	Output  string
	Parsed  *latexparser.ParseResult
}

// Compile compiles a note to PDF using latexmk
func (c *LatexmkCompiler) Compile(ctx context.Context, inputPath string, env []string) error {
	result := c.CompileWithOutput(ctx, inputPath, env)

	// If PDF was generated, consider it a success even if latexmk returned an error
	if result.Parsed.HasPDF {
		return nil
	}

	// If no PDF was generated and there are errors, return a formatted error
	if len(result.Parsed.Errors) > 0 {
		return fmt.Errorf("compilation failed with %d error(s)", len(result.Parsed.Errors))
	}

	// Fallback to original output if parsing didn't find anything
	if !result.Success {
		return fmt.Errorf("compilation failed:\n%s", result.Output)
	}

	return nil
}

// CompileWithOutput compiles and returns detailed output for better error reporting
func (c *LatexmkCompiler) CompileWithOutput(ctx context.Context, inputPath string, env []string) *CompileResult {
	// Validate input
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
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
		}
	}

	// Prepare latexmk command
	// -pdf: generate PDF using pdflatex
	// -interaction=nonstopmode: don't stop on errors, keep going
	// -file-line-error: better error messages with file:line: format
	// -halt-on-error: removed to allow continuation despite errors
	// -recorder: track file dependencies
	// -g: force rebuild (ignore timestamps)
	// -f: force completion even when errors occur
	args := []string{
		"-pdf",
		"-g",
		"-f",
		"-interaction=nonstopmode",
		"-file-line-error",
		"-recorder",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, "latexmk", args...)

	// Set working directory to notes path so latexmk can find .latexmkrc
	// The .latexmkrc file configures output to ../cache and TEXINPUTS paths
	cmd.Dir = c.vault.NotesPath

	// Prepare environment
	cmdEnv := os.Environ()

	// Add TEXINPUTS for template discovery
	// We append NotesPath and AssetsPath to TEXINPUTS so that any relative
	// imports that weren't caught by the preprocessor (or implicit ones) still work.
	texinputs := c.vault.GetTexInputsEnv()
	// Format: .:templates//:assets//:notes//:
	// Note: GetTexInputsEnv typically returns ".:templates//:" so we append to it
	texinputs = fmt.Sprintf("%s%s//:%s//:", texinputs, c.vault.AssetsPath, c.vault.NotesPath)

	cmdEnv = append(cmdEnv, "TEXINPUTS="+texinputs)
	cmdEnv = append(cmdEnv, env...)

	cmd.Env = cmdEnv

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Parse the output for meaningful errors/warnings
	parsed := latexparser.ParseLatexOutput(outputStr)

	return &CompileResult{
		Success: err == nil,
		Output:  outputStr,
		Parsed:  parsed,
	}
}

// GetOutputPath returns the path to the compiled PDF
func (c *LatexmkCompiler) GetOutputPath(slug string) string {
	// The preprocessor writes "slug.tex" to cache, so output is "slug.pdf" in the cache dir
	return c.vault.GetCachePath(slug + ".pdf")
}

// Clean removes auxiliary files for a specific note
func (c *LatexmkCompiler) Clean(ctx context.Context, slug string) error {
	// We target the preprocessed/cached file for cleaning
	filename := slug + ".tex"
	targetPath := c.vault.GetCachePath(filename)

	args := []string{
		"-C",
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
