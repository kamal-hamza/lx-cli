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
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
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
	Use:     "export [query]",
	Aliases: []string{"ex"},
	Short:   "Export a note to Markdown, HTML, or Docx (alias: ex)",
	Long: `Export a note to other formats using Pandoc.

This command:
1. Preprocesses the note (resolves links and paths).
2. Uses the asset repository to find images.
3. Converts the note to the desired format.

Examples:
  lx export "neural networks" -f markdown
  lx export graph -f html -o ./report.html
  lx export bayes -f docx -o ~/Downloads  (Saves as ~/Downloads/bayes-nets.docx)`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	// Defaults will be overridden in RunE if not changed
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Output format (markdown, html, docx)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output path (file or directory)")
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	query := args[0]

	// 0. Safety check
	if preprocessor == nil {
		return fmt.Errorf("internal error: preprocessor not initialized")
	}

	// 1. Use config default if flag not set
	if !cmd.Flags().Changed("format") {
		exportFormat = appConfig.DefaultExportFormat
	}

	// Validate Profile
	profile, ok := exportProfiles[exportFormat]
	if !ok {
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	// 2. Find the Note
	req := services.SearchRequest{Query: query}
	resp, err := listService.Search(ctx, req)
	if err != nil {
		return err
	}
	if resp.Total == 0 {
		return fmt.Errorf("no note found matching '%s'", query)
	}
	note := resp.Notes[0]

	fmt.Println(ui.FormatRocket(fmt.Sprintf("Exporting %s...", note.Title)))

	// 3. Setup Temp Filter
	tmpFilter, err := os.CreateTemp("", "lx-filter-*.lua")
	if err != nil {
		return fmt.Errorf("failed to create temp filter: %w", err)
	}
	defer os.Remove(tmpFilter.Name())
	if _, err := tmpFilter.WriteString(assets.LinksFilter); err != nil {
		return err
	}
	tmpFilter.Close()

	// 4. Determine Output Path
	destPath := exportOutput
	defaultFilename := fmt.Sprintf("%s.%s", note.Slug, profile.Extension)

	if destPath == "" {
		destPath = defaultFilename
	} else {
		// Check if provided path is a directory
		info, err := os.Stat(destPath)
		if err == nil && info.IsDir() {
			destPath = filepath.Join(destPath, defaultFilename)
		}
	}

	// 5. Convert
	if err := convertNote(note, filepath.Dir(destPath), tmpFilter.Name(), profile); err != nil {
		return err
	}

	// Preprocess
	sourcePath, err := preprocessor.Process(note.Slug)
	if err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	// Pandoc
	pandocArgs := []string{
		sourcePath,
		"-o", destPath,
		"--lua-filter", tmpFilter.Name(),
		"--resource-path=.:" + appVault.AssetsPath,
		"--standalone",
	}
	pandocArgs = append(pandocArgs, profile.PandocArgs...)

	// Metadata
	pandocArgs = append(pandocArgs, "--metadata", fmt.Sprintf("title=%s", note.Title))
	pandocArgs = append(pandocArgs, "--metadata", fmt.Sprintf("date=%s", note.Date))

	c := exec.Command("pandoc", pandocArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("pandoc failed: %w", err)
	}

	// Add frontmatter if needed (manual step for markdown)
	if profile.AddFrontmatter {
		content, _ := os.ReadFile(destPath)
		frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
slug: %s
tags: [%s]
---

`, note.Title, note.Date, note.Slug, strings.Join(note.Tags, ", "))
		os.WriteFile(destPath, append([]byte(frontmatter), content...), 0644)
	}

	fmt.Println(ui.FormatSuccess("Exported to: " + destPath))
	return nil
}

// convertNote exports a single note to a specific directory (Used by export-all)
func convertNote(h domain.NoteHeader, outDir, filterPath string, profile ExportProfile) error {
	// 1. Preprocess
	if preprocessor == nil {
		return fmt.Errorf("preprocessor not initialized")
	}
	sourcePath, err := preprocessor.Process(h.Slug)
	if err != nil {
		return err
	}

	// 2. Output Path
	destPath := filepath.Join(outDir, h.Slug+"."+profile.Extension)

	// 3. Pandoc Args
	args := []string{
		sourcePath,
		"-o", destPath,
		"--lua-filter", filterPath,
		"--resource-path=.:" + appVault.AssetsPath,
		"--standalone",
		"--metadata", fmt.Sprintf("title=%s", h.Title),
		"--metadata", fmt.Sprintf("date=%s", h.Date),
	}
	args = append(args, profile.PandocArgs...)

	// 4. Run
	cmd := exec.Command("pandoc", args...)
	// Capture output to buffer to avoid spamming stdout in concurrent mode
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc error on %s: %s", h.Slug, out.String())
	}

	// 5. Frontmatter
	if profile.AddFrontmatter {
		content, _ := os.ReadFile(destPath)
		frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
slug: %s
tags: [%s]
---

`, h.Title, h.Date, h.Slug, strings.Join(h.Tags, ", "))
		os.WriteFile(destPath, append([]byte(frontmatter), content...), 0644)
	}

	return nil
}
