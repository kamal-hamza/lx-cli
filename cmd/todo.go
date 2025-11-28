package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var todoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Interactive task manager",
	Long: `Aggregate and manage tasks from your notes.

Scans for:
  - \todo{...}   (LaTeX Package)
  - % TODO: ...  (Comments)

Controls:
  - ↑/↓   : Navigate
  - Enter : Open in Editor
  - Space : Toggle Done
  - q     : Quit`,
	RunE: runTodo,
}

// TodoItem represents a task found in a file
type TodoItem struct {
	ID       int
	Text     string
	Filename string
	LineNum  int
	Original string
	IsLatex  bool
}

func runTodo(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Scan for Tasks
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	var todos []TodoItem
	latexTodoRe := regexp.MustCompile(`\\todo\{([^}]+)\}`)
	commentTodoRe := regexp.MustCompile(`%\s*TODO:\s*(.*)`)

	fmt.Println(ui.FormatRocket("Scanning vault for tasks..."))

	count := 0
	for _, h := range headers {
		path := appVault.GetNotePath(h.Filename)
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// 1. Check LaTeX \todo{}
			if matches := latexTodoRe.FindStringSubmatch(trimmed); len(matches) > 1 {
				if strings.HasPrefix(trimmed, "%") {
					continue
				} // Skip commented out
				count++
				todos = append(todos, TodoItem{
					ID:       count,
					Text:     matches[1],
					Filename: h.Filename,
					LineNum:  lineNum,
					Original: line,
					IsLatex:  true,
				})
				continue
			}

			// 2. Check Comments % TODO:
			if matches := commentTodoRe.FindStringSubmatch(trimmed); len(matches) > 1 {
				count++
				todos = append(todos, TodoItem{
					ID:       count,
					Text:     matches[1],
					Filename: h.Filename,
					LineNum:  lineNum,
					Original: line,
					IsLatex:  false,
				})
			}
		}
		file.Close()
	}

	if len(todos) == 0 {
		fmt.Println(ui.FormatSuccess("Inbox Zero! No pending tasks found."))
		return nil
	}

	// 2. Start TUI
	p := tea.NewProgram(initialTodoModel(todos))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// --- TUI Model ---

type todoModel struct {
	table table.Model
	todos []TodoItem
}

func initialTodoModel(todos []TodoItem) todoModel {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Task", Width: 50},
		{Title: "File", Width: 20},
		{Title: "Line", Width: 6},
	}

	rows := []table.Row{}
	for _, t := range todos {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", t.ID),
			t.Text,
			safeTruncate(t.Filename, 20),
			fmt.Sprintf("%d", t.LineNum),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// --- Styles using Safe Terminal Colors ---
	s := table.DefaultStyles()

	// Header: Standard border color from UI package
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ui.ColorMuted).
		BorderBottom(true).
		Bold(true)

	// Selected: Use Primary color for background, Default (White/Light) for text
	// This uses standard ANSI colors (5 and 7) which are safe on all terminals.
	s.Selected = s.Selected.
		Foreground(ui.ColorDefault).
		Background(ui.ColorPrimary).
		Bold(true)

	t.SetStyles(s)

	return todoModel{
		table: t,
		todos: todos,
	}
}

func (m todoModel) Init() tea.Cmd { return nil }

func (m todoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			idx := m.table.Cursor()
			if idx < len(m.todos) {
				target := m.todos[idx]
				return m, func() tea.Msg {
					path := appVault.GetNotePath(target.Filename)
					// Use the shared helper from utils.go
					OpenEditorAtLine(path, target.LineNum)
					return tea.Quit
				}
			}

		case " ", "x":
			idx := m.table.Cursor()
			if idx < len(m.todos) {
				target := m.todos[idx]
				markTaskDone(target)

				// Remove row safely
				if len(m.todos) > 0 {
					m.todos = append(m.todos[:idx], m.todos[idx+1:]...)
					m.table.SetRows(removeRow(m.table.Rows(), idx))
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m todoModel) View() string {
	if len(m.todos) == 0 {
		return "\n  ✅ All tasks completed!\n\n  Press 'q' to quit.\n"
	}

	return "\n" +
		ui.StyleTitle.Render(" ✅ Task Dashboard ") + "\n\n" +
		m.table.View() + "\n\n" +
		ui.FormatMuted(" [Space] Mark Done  [Enter] Open Note  [q] Quit") + "\n"
}

// --- Helpers ---

func safeTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func markTaskDone(t TodoItem) {
	path := appVault.GetNotePath(t.Filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	if t.LineNum > len(lines) {
		return
	}

	original := lines[t.LineNum-1]
	var replacement string

	if t.IsLatex {
		re := regexp.MustCompile(`\\todo\{([^}]+)\}`)
		replacement = re.ReplaceAllString(original, "% DONE: $1")
	} else {
		replacement = strings.Replace(original, "TODO:", "DONE:", 1)
	}

	lines[t.LineNum-1] = replacement
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func removeRow(rows []table.Row, i int) []table.Row {
	return append(rows[:i], rows[i+1:]...)
}
