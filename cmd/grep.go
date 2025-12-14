package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var grepCmd = &cobra.Command{
	Use:     "grep",
	Aliases: []string{"g"},
	Short:   "Interactive vault search (alias: g)",
	Long: `Search through all notes using a fuzzy finder.

This scans all lines in your vault and lets you filter them interactively.
Select a result to open the note at that specific line.

Examples:
  lx grep`,
	Args: cobra.NoArgs,
	RunE: runGrep,
}

func runGrep(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	fmt.Println(ui.FormatRocket("Scanning vault..."))

	// 1. Get ALL lines from ALL notes
	// Passing empty string "" triggers "load all" mode
	matches, err := grepService.Execute(ctx, "")
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		fmt.Println(ui.FormatWarning("No notes found to search."))
		return nil
	}

	// 2. Launch Fuzzy Finder
	idx, err := fuzzyfinder.Find(
		matches,
		func(i int) string {
			m := matches[i]
			// Display: "slug:line  content"
			return fmt.Sprintf("%s:%d  %s",
				ui.StyleMuted.Render(m.Slug),
				m.LineNum,
				m.Content)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			m := matches[i]
			return generatePreview(appVault.GetNotePath(m.Filename), m.LineNum)
		}),
	)

	if err != nil {
		// User cancelled
		fmt.Println(ui.FormatInfo("Search cancelled."))
		return nil
	}

	// 3. Open Editor
	selected := matches[idx]
	notePath := appVault.GetNotePath(selected.Filename)

	fmt.Printf("Opening %s at line %d...\n", selected.Slug, selected.LineNum)
	return openEditorAtLine(notePath, selected.LineNum)
}

// generatePreview reads the file on-demand to show context around the match
func generatePreview(path string, targetLine int) string {
	file, err := os.Open(path)
	if err != nil {
		return "Failed to read file"
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)

	// Configuration
	contextLines := 3
	startLine := targetLine - contextLines
	endLine := targetLine + contextLines

	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine < startLine {
			continue
		}
		if currentLine > endLine {
			break
		}

		text := scanner.Text()
		prefix := "  "
		style := ui.StyleMuted

		if currentLine == targetLine {
			prefix = ">>"
			style = ui.StyleBold
		}

		sb.WriteString(style.Render(fmt.Sprintf("%s %s", prefix, text)) + "\n")
	}

	return sb.String()
}

func openEditorAtLine(path string, line int) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	var args []string

	// VS Code / Sublime / generic GUI editors often use file:line syntax
	if strings.Contains(editor, "code") || strings.Contains(editor, "subl") {
		if strings.Contains(editor, "code") {
			// VS Code specific: code -g file:line
			args = []string{"-g", fmt.Sprintf("%s:%d", path, line)}
		} else {
			args = []string{fmt.Sprintf("%s:%d", path, line)}
		}
	} else {
		// Terminal editors (vim, nano) usually use +line
		args = []string{fmt.Sprintf("+%d", line), path}
	}

	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
