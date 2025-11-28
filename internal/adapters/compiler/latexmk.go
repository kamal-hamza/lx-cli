package compiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"lx/pkg/vault"
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
func (c *LatexmkCompiler) Compile(ctx context.Context, slug string, env []string) error {
	// Find the source file
	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		return err
	}

	sourcePath := c.vault.GetNotePath(sourceFile)

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
		sourcePath,
	}

	cmd := exec.CommandContext(ctx, "latexmk", args...)

	// Set working directory to cache
	cmd.Dir = c.vault.CachePath

	// Prepare environment
	cmdEnv := os.Environ()

	// Add TEXINPUTS for template discovery
	texinputs := c.vault.GetTexInputsEnv()
	cmdEnv = append(cmdEnv, "TEXINPUTS="+texinputs)

	// Add any additional environment variables
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
	// Find the source file to get the base name
	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		// Fallback to constructing from slug
		return c.vault.GetCachePath(slug + ".pdf")
	}

	// Replace .tex with .pdf
	pdfName := strings.TrimSuffix(sourceFile, ".tex") + ".pdf"
	return c.vault.GetCachePath(pdfName)
}

// findSourceFile finds the .tex file matching the slug
func (c *LatexmkCompiler) findSourceFile(slug string) (string, error) {
	entries, err := os.ReadDir(c.vault.NotesPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tex") {
			continue
		}

		// Check if filename contains the slug
		name := strings.TrimSuffix(entry.Name(), ".tex")
		if strings.HasSuffix(name, slug) || strings.Contains(name, "-"+slug) {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("source file not found for slug: %s", slug)
}

// Clean removes auxiliary files for a specific note
func (c *LatexmkCompiler) Clean(ctx context.Context, slug string) error {
	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		return err
	}

	sourcePath := c.vault.GetNotePath(sourceFile)

	args := []string{
		"-C",
		"-output-directory=" + c.vault.CachePath,
		sourcePath,
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
