package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/assets"
	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	exportFormat    string
	exportOutputDir string
)

// ExportProfile defines how to handle different output formats
type ExportProfile struct {
	Extension      string
	PandocArgs     []string
	AddFrontmatter bool
}

var exportProfiles = map[string]ExportProfile{
	"markdown": {
		Extension: "md",
		PandocArgs: []string{
			"-f", "latex",
			"-t", "gfm",
			"--wrap=none",
			"--mathjax",
		},
		AddFrontmatter: true,
	},
	"html": {
		Extension: "html",
		PandocArgs: []string{
			"-f", "latex",
			"-t", "html",
			"--standalone",
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
	Use:   "export",
	Short: "Export vault using Pandoc",
	Long: `Export your notes to other formats using Pandoc.

By default, files are exported to the 'exports/' directory in your vault.
You can specify a custom directory using the --output flag.

Supported Formats:
  - markdown (Default): Obsidian-compatible.
  - html: Standalone HTML5.
  - docx: Microsoft Word.

Examples:
  lx export                   # Export to <vault>/exports
  lx export -o ~/Desktop/dist # Export to custom folder
  lx export -f docx           # Export as Word docs`,
	Args: cobra.NoArgs,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Output format (markdown, html, docx)")
	exportCmd.Flags().StringVarP(&exportOutputDir, "output", "o", "", "Custom output directory (default: vault/exports)")
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Validate Dependencies
	if err := checkAndInstallPandoc(); err != nil {
		return fmt.Errorf("pandoc check failed: %w", err)
	}

	// 2. Validate Profile
	profile, ok := exportProfiles[exportFormat]
	if !ok {
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	// 3. Determine Output Directory
	outDir := exportOutputDir
	if outDir == "" {
		// Default: <vault_root>/exports
		outDir = filepath.Join(appVault.RootPath, "exports")
	}

	// Ensure absolute path for clarity in logs
	if abs, err := filepath.Abs(outDir); err == nil {
		outDir = abs
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// 4. Create Temporary Lua Filter
	tmpFile, err := os.CreateTemp("", "lx-filter-*.lua")
	if err != nil {
		return fmt.Errorf("failed to create temp filter: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(assets.LinksFilter); err != nil {
		return err
	}
	tmpFile.Close()

	// 5. Get Notes
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Exporting %d notes to %s...", len(headers), outDir)))

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

	args := append([]string{}, profile.PandocArgs...)
	args = append(args, "--lua-filter", filterPath)

	// Add resource path so Pandoc finds images in assets/
	args = append(args, "--resource-path", appVault.AssetsPath)

	args = append(args, "--metadata", fmt.Sprintf("title=%s", h.Title))
	args = append(args, "--metadata", fmt.Sprintf("date=%s", h.Date))
	args = append(args, srcPath)

	cmd := exec.Command("pandoc", args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc error: %w", err)
	}

	result := out.String()

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

	return os.WriteFile(destPath, []byte(result), 0644)
}
