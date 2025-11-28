package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var exploreCmd = &cobra.Command{
	Use:   "explore [start-note]",
	Short: "Interactive graph explorer",
	Long: `Maps your knowledge graph interactively.

Vim Navigation:
- k / ↑ : Move Up
- j / ↓ : Move Down
- l / → : Go Forward (Select Node)
- h / ← : Go Back
- o     : Open in Editor
- q     : Quit`,
	RunE: runExplore,
}

func runExplore(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Initialize Service & Fetch Data
	graphSvc := services.NewGraphService(noteRepo, appVault.RootPath)
	data, err := graphSvc.GetGraph(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to load graph: %w", err)
	}

	if len(data.Nodes) == 0 {
		fmt.Println(ui.FormatWarning("Graph is empty."))
		return nil
	}

	// 2. Build Adjacency Map
	adjacency := make(map[string][]string)
	titles := make(map[string]string)

	for _, n := range data.Nodes {
		titles[n.ID] = n.Title
	}

	for _, l := range data.Links {
		adjacency[l.Source] = append(adjacency[l.Source], l.Target)
		adjacency[l.Target] = append(adjacency[l.Target], l.Source)
	}

	for k, v := range adjacency {
		adjacency[k] = unique(v)
	}

	// 3. Determine Start Node
	startSlug := data.Nodes[0].ID
	if len(args) > 0 {
		query := strings.ToLower(args[0])
		for _, n := range data.Nodes {
			if strings.Contains(strings.ToLower(n.Title), query) || strings.Contains(n.ID, query) {
				startSlug = n.ID
				break
			}
		}
	}

	// 4. Run TUI
	p := tea.NewProgram(initialModel(startSlug, titles, adjacency))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// --- TUI Model ---

type model struct {
	currentSlug string
	titles      map[string]string
	adjacency   map[string][]string
	cursor      int
	neighbors   []string
	history     []string // Stack to track navigation history
}

func initialModel(start string, titles map[string]string, adj map[string][]string) model {
	m := model{
		currentSlug: start,
		titles:      titles,
		adjacency:   adj,
		cursor:      0,
		history:     []string{},
	}
	m.updateNeighbors()
	return m
}

func (m *model) updateNeighbors() {
	nbs := m.adjacency[m.currentSlug]
	sort.Strings(nbs)
	m.neighbors = nbs
	m.cursor = 0
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		// --- VIM NAVIGATION ---

		// Up (k)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// Down (j)
		case "down", "j":
			if m.cursor < len(m.neighbors)-1 {
				m.cursor++
			}

		// Forward (l / Enter)
		case "right", "l", "enter":
			if len(m.neighbors) > 0 {
				// 1. Push current node to history
				m.history = append(m.history, m.currentSlug)

				// 2. Move to new node
				m.currentSlug = m.neighbors[m.cursor]
				m.updateNeighbors()
			}

		// Back (h / Left)
		case "left", "h":
			if len(m.history) > 0 {
				// 1. Pop last node from history
				lastIdx := len(m.history) - 1
				previousSlug := m.history[lastIdx]
				m.history = m.history[:lastIdx] // Remove from stack

				// 2. Go back
				m.currentSlug = previousSlug
				m.updateNeighbors()
			}

		case "o":
			return m, openEditorCmd(m.currentSlug)
		}
	}
	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	title := m.titles[m.currentSlug]

	// Header
	s.WriteString("\n")
	s.WriteString(ui.StyleTitle.Render(" 🧠 " + title))
	s.WriteString(ui.StyleMuted.Render(fmt.Sprintf(" (%s)", m.currentSlug)))
	s.WriteString("\n\n")

	// Breadcrumbs / History Hint
	if len(m.history) > 0 {
		prev := m.titles[m.history[len(m.history)-1]]
		if prev == "" {
			prev = m.history[len(m.history)-1]
		}
		s.WriteString(ui.StyleMuted.Render(fmt.Sprintf(" < Back to: %s (h)", prev)))
		s.WriteString("\n\n")
	}

	// Connections List
	s.WriteString(ui.StyleBold.Render(fmt.Sprintf(" Connections (%d)", len(m.neighbors))))
	s.WriteString("\n\n")

	if len(m.neighbors) == 0 {
		s.WriteString(ui.StyleMuted.Render("  (No connections found)"))
	} else {
		for i, slug := range m.neighbors {
			cursor := "  "
			style := ui.StyleMuted

			if m.cursor == i {
				cursor = ui.StyleAccent.Render("→ ")
				style = ui.StyleSuccess.Copy().Bold(true)
			}

			neighborTitle := m.titles[slug]
			if neighborTitle == "" {
				neighborTitle = slug
			}

			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(neighborTitle)))
		}
	}

	// Footer help
	s.WriteString("\n\n")
	s.WriteString(ui.StyleMuted.Render(" [k/j] Navigate  [l] Enter  [h] Back  [o] Open  [q] Quit"))
	s.WriteString("\n")

	return s.String()
}

func openEditorCmd(slug string) tea.Cmd {
	return func() tea.Msg {
		return tea.Quit
	}
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
