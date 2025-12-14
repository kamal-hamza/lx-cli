package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var renameTemplate bool

var renameCmd = &cobra.Command{
	Use:     "rename [query] [new-title]",
	Aliases: []string{"mv"},
	Short:   "Rename a note and update references (alias: mv)",
	Long: `Rename a note and update all backlinks and imports.

Refactors:
- \ref{old-slug}     -> \ref{new-slug}
- \input{old-slug}   -> \input{new-slug}
- \include{old-slug} -> \include{new-slug}
- \cite{old-slug}    -> \cite{new-slug}

Examples:
  lx rename graph "Graph Theory"
  lx rename -t homework "problem-set"`,
	Args: cobra.MaximumNArgs(2),
	RunE: runRename,
}

func init() {
	renameCmd.Flags().BoolVarP(&renameTemplate, "template", "t", false, "Rename a template instead of a note")
}

func runRename(cmd *cobra.Command, args []string) error {
	if renameTemplate {
		return runRenameTemplate(cmd, args)
	}
	return runRenameNote(cmd, args)
}

func runRenameNote(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Parse Arguments & Select Note
	var query string
	var newTitle string

	if len(args) > 0 {
		query = args[0]
	}
	if len(args) > 1 {
		newTitle = args[1]
	}

	var target *domain.NoteHeader

	if query == "" {
		// Interactive Selection
		req := services.ListRequest{SortBy: "date"}
		resp, err := listService.Execute(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			fmt.Println(ui.FormatWarning("No notes found."))
			return nil
		}

		idx, err := fuzzyfinder.Find(
			resp.Notes,
			func(i int) string { return resp.Notes[i].Title },
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				return fmt.Sprintf("Rename Note\n\nTitle: %s\nSlug: %s", resp.Notes[i].Title, resp.Notes[i].Slug)
			}),
		)
		if err != nil {
			return nil
		}
		target = &resp.Notes[idx]
	} else {
		// Search by query
		req := services.SearchRequest{Query: query}
		resp, err := listService.Search(ctx, req)
		if err != nil {
			return err
		}
		if resp.Total == 0 {
			return fmt.Errorf("no note found matching '%s'", query)
		}
		target = &resp.Notes[0]
	}

	// 2. Prompt for New Title (if not provided)
	if newTitle == "" {
		fmt.Println(ui.FormatInfo("Selected: " + target.Title))
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(ui.StyleInfo.Render("Enter new title: "))
		input, _ := reader.ReadString('\n')
		newTitle = strings.TrimSpace(input)
		if newTitle == "" {
			return fmt.Errorf("title cannot be empty")
		}
	}

	oldSlug := target.Slug

	// 3. Perform Rename
	fmt.Println(ui.FormatInfo(fmt.Sprintf("Renaming '%s' -> '%s'...", target.Title, newTitle)))
	if err := noteRepo.Rename(ctx, oldSlug, newTitle); err != nil {
		return err
	}

	// Calculate new slug to perform refactoring
	// (We re-generate it using the same logic as the repo to ensure consistency)
	// Ideally we'd get this from the repo response, but for now this works.
	newSlug := domain.GenerateSlug(newTitle)

	// 4. Smart Refactor
	fmt.Println(ui.FormatRocket("Refactoring references..."))

	matches, err := grepService.Execute(ctx, oldSlug)
	if err != nil {
		return err
	}

	filesToEdit := make(map[string]bool)
	for _, m := range matches {
		if m.Slug != newSlug {
			filesToEdit[m.Filename] = true
		}
	}

	count := 0
	refRegex := regexp.MustCompile(`\\(ref|input|include|cite)\{` + regexp.QuoteMeta(oldSlug) + `\}`)

	for filename := range filesToEdit {
		path := appVault.GetNotePath(filename)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		newContent := refRegex.ReplaceAllString(string(content), `\$1{`+newSlug+`}`)

		if newContent != string(content) {
			if err := os.WriteFile(path, []byte(newContent), 0644); err == nil {
				fmt.Printf("  %s %s\n", ui.FormatSuccess("Updated"), filename)
				count++
			}
		}
	}

	if count == 0 {
		fmt.Println(ui.FormatMuted("No incoming references found."))
	} else {
		fmt.Println(ui.FormatSuccess(fmt.Sprintf("Updated references in %d files.", count)))
	}

	return nil
}

func runRenameTemplate(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Select Template
	var query, newName string
	if len(args) > 0 {
		query = args[0]
	}
	if len(args) > 1 {
		newName = args[1]
	}

	templates, err := templateRepo.List(ctx)
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		fmt.Println(ui.FormatWarning("No templates found."))
		return nil
	}

	var selected *domain.Template

	if query == "" {
		idx, err := fuzzyfinder.Find(
			templates,
			func(i int) string { return templates[i].Name },
		)
		if err != nil {
			return nil
		}
		selected = &templates[idx]
	} else {
		// Simple search
		for _, t := range templates {
			if strings.Contains(t.Name, query) {
				selected = &t
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("template not found: %s", query)
		}
	}

	// 2. Prompt for New Name
	if newName == "" {
		fmt.Println(ui.FormatInfo("Selected: " + selected.Name))
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(ui.StyleInfo.Render("Enter new template name (no extension): "))
		input, _ := reader.ReadString('\n')
		newName = strings.TrimSpace(input)
		if newName == "" {
			return fmt.Errorf("name cannot be empty")
		}
	}

	// 3. Rename File
	oldPath := selected.Path
	newFilename := newName
	if !strings.HasSuffix(newFilename, ".sty") {
		newFilename += ".sty"
	}
	newPath := appVault.GetTemplatePath(newFilename)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename template: %w", err)
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Renamed template to: %s", newName)))
	return nil
}
