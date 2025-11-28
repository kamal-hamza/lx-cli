package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"lx/internal/assets"
	"lx/internal/core/domain"
	"lx/pkg/ui"

	"github.com/spf13/cobra"
)

// --- Configuration ---

var exportFormat string

// ExportProfile defines how to handle different output formats
type ExportProfile struct {
	Extension      string
	PandocArgs     []string
	AddFrontmatter bool // Should we prepend YAML frontmatter? (obsidian style)
}

// Registry of supported formats
var exportProfiles = map[string]ExportProfile{
	"markdown": {
		Extension: "md",
		PandocArgs: []string{
			"-f", "latex",
			"-t", "gfm", // GitHub Flavored Markdown
			"--wrap=none", // Don't hard wrap lines
			"--mathjax",   // Preserve math for web/obsidian
		},
		AddFrontmatter: true,
	},
	"html": {
		Extension: "html",
		PandocArgs: []string{
			"-f", "latex",
			"-t", "html",
			"--standalone", // Full HTML document with <head>
			"--mathjax",
		},
		AddFrontmatter: false,
	},
	"docx": {
		Extension: "docx",
		PandocArgs: []string{
			"-f", "latex",
			"-t", "docx",
		},
		AddFrontmatter: false,
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [output-dir]",
	Short: "Export vault using Pandoc (Markdown, HTML, Docx)",
	Long: `Export your notes to other formats using Pandoc with custom Lua filters.

Supported Formats:
  - markdown (Default): Obsidian-compatible with WikiLinks [[...]] and YAML Frontmatter.
  - html: Standalone HTML5 with MathJax.
  - docx: Microsoft Word document.

Examples:
  lx export                   # Export to ./dist (Markdown)
  lx export -f html ./web     # Export to HTML
  lx export -f docx ./docs    # Export to Word`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Output format (markdown, html, docx)")
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Validate Dependencies
	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf("pandoc not found: please install it to use export")
	}

	// 2. Validate Profile
	profile, ok := exportProfiles[exportFormat]
	if !ok {
		return fmt.Errorf("unsupported format: %s (valid: markdown, html, docx)", exportFormat)
	}

	// 3. Setup Directories
	outDir := "dist"
	if len(args) > 0 {
		outDir = args[0]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// 4. Create Temporary Lua Filter File
	tmpFile, err := os.CreateTemp("", "lx-filter-*.lua")
	if err != nil {
		return fmt.Errorf("failed to create temp filter: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the embedded content to the temp file
	if _, err := tmpFile.WriteString(assets.LinksFilter); err != nil {
		return err
	}
	tmpFile.Close()
	// 5. Get All Notes
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Exporting %d notes to %s (%s)...", len(headers), outDir, exportFormat)))

	// 6. Process Notes
	success := 0
	for _, h := range headers {
		if err := convertNote(h, outDir, tmpFile.Name(), profile); err != nil {
			fmt.Println(ui.FormatWarning(fmt.Sprintf("Failed %s: %v", h.Slug, err)))
			continue
		}
		success++
		if success%10 == 0 {
			fmt.Print(".")
		}
	}
	fmt.Println()
	fmt.Println(ui.FormatSuccess("Export complete!"))

	return nil
}

func convertNote(h domain.NoteHeader, outDir, filterPath string, profile ExportProfile) error {
	srcPath := appVault.GetNotePath(h.Filename)
	destPath := filepath.Join(outDir, h.Slug+"."+profile.Extension)

	// Build Pandoc Command
	// Start with profile args
	args := append([]string{}, profile.PandocArgs...)

	// Add Lua Filter
	args = append(args, "--lua-filter", filterPath)

	// Add Metadata (Title/Date) for non-markdown formats (HTML/Docx need this for document properties)
	args = append(args, "--metadata", fmt.Sprintf("title=%s", h.Title))
	args = append(args, "--metadata", fmt.Sprintf("date=%s", h.Date))

	// Input file
	args = append(args, srcPath)

	// Execute Pandoc
	cmd := exec.Command("pandoc", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	// cmd.Stderr = os.Stderr // Uncomment for debugging

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc error: %w", err)
	}

	result := out.String()

	// Post-Processing: YAML Frontmatter
	// Only for Markdown (Obsidian needs this to recognize title/tags)
	if profile.AddFrontmatter {
		frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
slug: %s
tags: [%s]
---

`, h.Title, h.Date, h.Slug, strings.Join(h.Tags, ", "))

		result = frontmatter + result
	}

	// Write Output
	return os.WriteFile(destPath, []byte(result), 0644)
}
