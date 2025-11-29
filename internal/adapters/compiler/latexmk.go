package compiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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

// Compile compiles a note to PDF using latexmk
func (c *LatexmkCompiler) Compile(ctx context.Context, inputPath string, env []string) error {
	// Validate input
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", inputPath)
	}

	// Prepare latexmk command
	// -pdf: generate PDF using pdflatex
	// -output-directory: where to put output files
	// -interaction=nonstopmode: don't stop on errors
	// -file-line-error: better error messages
	args := []string{
		"-pdf",
		"-output-directory=" + c.vault.CachePath,
		"-interaction=nonstopmode",
		"-file-line-error",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, "latexmk", args...)

	// Set working directory to cache (since inputPath is usually in cache)
	cmd.Dir = c.vault.CachePath

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
	if err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
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
		"-output-directory=" + c.vault.CachePath,
		targetPath,
	}

	cmd := exec.CommandContext(ctx, "latexmk", args...)
	cmd.Dir = c.vault.CachePath

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
