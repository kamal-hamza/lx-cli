package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Launch interactive dashboard (alias: dash)",
	Long: `Launch a full-screen interactive dashboard for managing notes.

The dashboard provides:
- List view with all notes sorted by modification time
- Real-time search and filtering
- Quick actions: open, edit, build, delete, create
- Graph view for visualizing note connections

Keyboard Shortcuts:
  Navigation:
    ↑/k         Move up
    ↓/j         Move down
    g           Jump to top
    G           Jump to bottom

  Actions:
    Enter       Open note (PDF)
    e           Edit note (source)
    b           Build note
    d           Delete note
    n           Create new note

  Views:
    /           Search mode
    Esc         Clear search / Exit mode
    v           Toggle graph view
    ?           Show help

  General:
    q           Quit dashboard
    Ctrl+C      Force quit`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// Load initial data
	listResp, err := listService.Execute(ctx, services.ListRequest{
		SortBy:  "date",
		Reverse: true,
	})
	if err != nil {
		return fmt.Errorf("failed to load notes: %w", err)
	}

	// Initialize dashboard model
	m := newDashboardModel(ctx, listResp.Notes)

	// Run the TUI
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running dashboard: %w", err)
	}

	return nil
}

// Dashboard view modes
type viewMode int

const (
	modeList viewMode = iota
	modeSearch
	modeGraph
	modeHelp
	modeConfirmDelete
)

// Preview state
type previewState struct {
	content  string
	slug     string
	viewport viewport.Model
}

// Dashboard model
type dashboardModel struct {
	ctx           context.Context
	notes         []domain.NoteHeader // All notes
	filteredNotes []domain.NoteHeader // Filtered/searched notes
	cursor        int                 // Selected item index
	offset        int                 // Scroll offset for viewport
	mode          viewMode
	searchInput   textinput.Model
	help          help.Model
	keys          keyMap
	width         int
	height        int
	ready         bool
	message       string // Status message
	messageStyle  lipgloss.Style
	messageExpiry time.Time
	deleteTarget  *domain.NoteHeader // Note pending deletion
	graphData     *services.GraphData
	graphCursor   int
	graphHistory  []string
	preview       previewState
}

// Key bindings
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Open    key.Binding
	Edit    key.Binding
	Build   key.Binding
	Delete  key.Binding
	New     key.Binding
	Search  key.Binding
	Graph   key.Binding
	Help    key.Binding
	Quit    key.Binding
	Escape  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Preview key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Edit, k.Search, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Open, k.Edit, k.Build, k.Delete, k.New},
		{k.Search, k.Preview, k.Graph, k.Help, k.Escape, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter", "o"),
		key.WithHelp("enter/o", "open PDF"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit source"),
	),
	Build: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "build"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new note"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Graph: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "graph view"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "N", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
	Preview: key.NewBinding(
		key.WithKeys(""),
		key.WithHelp("", ""),
	),
}

func newDashboardModel(ctx context.Context, notes []domain.NoteHeader) dashboardModel {
	ti := textinput.New()
	ti.Placeholder = "Search notes..."
	ti.CharLimit = 100
	ti.Width = 50

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Foreground(ui.ColorDefault)

	return dashboardModel{
		ctx:           ctx,
		notes:         notes,
		filteredNotes: notes,
		cursor:        0,
		offset:        0,
		mode:          modeList,
		searchInput:   ti,
		help:          help.New(),
		keys:          keys,
		ready:         false,
		preview: previewState{
			viewport: vp,
		},
	}
}

func (m dashboardModel) Init() tea.Cmd {
	// Load preview for the first note
	if len(m.notes) > 0 {
		return m.loadPreview(m.notes[0])
	}
	return nil
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true

		// Update preview viewport size
		previewWidth := (msg.Width / 2) - 4
		previewHeight := msg.Height - 16
		if previewHeight < 10 {
			previewHeight = 10
		}
		m.preview.viewport.Width = previewWidth
		m.preview.viewport.Height = previewHeight
		return m, nil

	case tea.KeyMsg:
		// Handle mode-specific key bindings
		switch m.mode {
		case modeSearch:
			return m.updateSearch(msg)
		case modeHelp:
			return m.updateHelp(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		case modeGraph:
			return m.updateGraph(msg)
		case modeList:
			return m.updateList(msg)
		}

	case statusMsg:
		m.message = msg.message
		m.messageStyle = msg.style
		m.messageExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case clearMessageMsg:
		if time.Now().After(m.messageExpiry) {
			m.message = ""
		}
		return m, nil

	case reloadNotesMsg:
		// Reload notes from disk
		listResp, err := listService.Execute(m.ctx, services.ListRequest{
			SortBy:  "date",
			Reverse: true,
		})
		if err == nil {
			m.notes = listResp.Notes
			m.applySearch()
			// Reload preview
			if len(m.filteredNotes) > 0 {
				return m, m.loadPreview(m.filteredNotes[m.cursor])
			}
		}
		return m, nil

	case previewLoadedMsg:
		m.preview.content = msg.content
		m.preview.slug = msg.slug
		m.preview.viewport.SetContent(msg.content)
		m.preview.viewport.GotoTop()
		return m, nil

	case loadGraphMsg:
		m.graphData = msg.data
		m.mode = modeGraph
		m.graphCursor = 0
		if len(m.filteredNotes) > 0 {
			m.graphHistory = []string{m.filteredNotes[m.cursor].Slug}
		}
		return m, nil
	}

	// Update viewport if we're in list or search mode (preview is always visible)
	if m.mode == modeList || m.mode == modeSearch {
		var cmd tea.Cmd
		m.preview.viewport, cmd = m.preview.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m dashboardModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.adjustViewport()
			if len(m.filteredNotes) > 0 {
				return m, m.loadPreview(m.filteredNotes[m.cursor])
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredNotes)-1 {
			m.cursor++
			m.adjustViewport()
			if len(m.filteredNotes) > 0 {
				return m, m.loadPreview(m.filteredNotes[m.cursor])
			}
		}

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		m.offset = 0
		if len(m.filteredNotes) > 0 {
			return m, m.loadPreview(m.filteredNotes[m.cursor])
		}

	case key.Matches(msg, m.keys.Bottom):
		m.cursor = len(m.filteredNotes) - 1
		m.adjustViewport()
		if len(m.filteredNotes) > 0 {
			return m, m.loadPreview(m.filteredNotes[m.cursor])
		}

	case msg.Type == tea.KeyPgUp:
		m.preview.viewport.ViewUp()

	case msg.Type == tea.KeyPgDown:
		m.preview.viewport.ViewDown()

	case key.Matches(msg, m.keys.Open):
		if len(m.filteredNotes) > 0 {
			return m, m.openNote(m.filteredNotes[m.cursor])
		}

	case key.Matches(msg, m.keys.Edit):
		if len(m.filteredNotes) > 0 {
			return m, m.editNote(m.filteredNotes[m.cursor])
		}

	case key.Matches(msg, m.keys.Build):
		if len(m.filteredNotes) > 0 {
			return m, m.buildNote(m.filteredNotes[m.cursor])
		}

	case key.Matches(msg, m.keys.Delete):
		if len(m.filteredNotes) > 0 {
			m.deleteTarget = &m.filteredNotes[m.cursor]
			m.mode = modeConfirmDelete
		}

	case key.Matches(msg, m.keys.New):
		return m, m.createNote()

	case key.Matches(msg, m.keys.Search):
		m.mode = modeSearch
		m.searchInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Graph):
		return m, m.loadGraph()

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
	}

	return m, nil
}

func (m dashboardModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = modeList
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filteredNotes = m.notes
		m.cursor = 0
		m.offset = 0
		return m, nil

	// Enter key to open note from search
	case msg.Type == tea.KeyEnter:
		if len(m.filteredNotes) > 0 {
			m.mode = modeList
			m.searchInput.Blur()
			return m, m.openNote(m.filteredNotes[m.cursor])
		}

	// Only use arrow keys for navigation in search mode, not j/k
	case msg.Type == tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
			m.adjustViewport()
			if len(m.filteredNotes) > 0 {
				return m, m.loadPreview(m.filteredNotes[m.cursor])
			}
		}

	case msg.Type == tea.KeyDown:
		if m.cursor < len(m.filteredNotes)-1 {
			m.cursor++
			m.adjustViewport()
			if len(m.filteredNotes) > 0 {
				return m, m.loadPreview(m.filteredNotes[m.cursor])
			}
		}

	case msg.Type == tea.KeyPgUp:
		m.preview.viewport.ViewUp()

	case msg.Type == tea.KeyPgDown:
		m.preview.viewport.ViewDown()

	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
		oldQuery := m.searchInput.Value()
		m.applySearch()
		// Reload preview if search changed
		if len(m.filteredNotes) > 0 && m.searchInput.Value() != oldQuery {
			return m, tea.Batch(cmd, m.loadPreview(m.filteredNotes[m.cursor]))
		}
		return m, cmd
	}

	return m, nil
}

func (m dashboardModel) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Quit):
		m.mode = modeList
	}
	return m, nil
}

func (m dashboardModel) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		note := m.deleteTarget
		m.deleteTarget = nil
		m.mode = modeList
		return m, m.deleteNoteConfirmed(note)

	case key.Matches(msg, m.keys.Cancel):
		m.deleteTarget = nil
		m.mode = modeList
	}
	return m, nil
}

func (m dashboardModel) updateGraph(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Graph):
		m.mode = modeList
		m.graphData = nil
		m.graphHistory = nil

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.graphCursor > 0 {
			m.graphCursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.graphData != nil {
			neighbors := m.getGraphNeighbors()
			if m.graphCursor < len(neighbors)-1 {
				m.graphCursor++
			}
		}

	case key.Matches(msg, m.keys.Open):
		if m.graphData != nil {
			neighbors := m.getGraphNeighbors()
			if len(neighbors) > 0 && m.graphCursor < len(neighbors) {
				// Navigate to selected node
				m.graphHistory = append(m.graphHistory, m.getCurrentGraphNode())
				selectedNode := neighbors[m.graphCursor]
				m.setGraphNode(selectedNode)
			}
		}

	case msg.String() == "h", msg.String() == "left":
		// Go back in graph history
		if len(m.graphHistory) > 0 {
			lastIdx := len(m.graphHistory) - 1
			previousNode := m.graphHistory[lastIdx]
			m.graphHistory = m.graphHistory[:lastIdx]
			m.setGraphNode(previousNode)
		}
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if !m.ready {
		return "\n  Loading dashboard..."
	}

	switch m.mode {
	case modeHelp:
		return m.viewHelp()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	case modeGraph:
		return m.viewGraph()
	default:
		return m.viewList()
	}
}

func (m dashboardModel) viewList() string {
	// Always show preview
	return m.viewListWithPreview()
}

func (m dashboardModel) viewListOld() string {
	var s strings.Builder

	// Header
	header := m.renderHeader()
	s.WriteString(header)
	s.WriteString("\n")

	// Search bar (always visible, highlighted when active)
	searchBar := m.renderSearchBar()
	s.WriteString(searchBar)
	s.WriteString("\n\n")

	// Notes list
	notesList := m.renderNotesList()
	s.WriteString(notesList)

	// Footer
	footer := m.renderFooter()
	s.WriteString("\n")
	s.WriteString(footer)

	return s.String()
}

func (m dashboardModel) viewListWithPreview() string {
	// Split screen: list on left (40%), preview on right (60%)
	listWidth := int(float64(m.width) * 0.4)
	previewWidth := m.width - listWidth - 2 // -2 for border

	if listWidth < 30 {
		listWidth = 30
	}
	if previewWidth < 40 {
		// Screen too narrow, fall back to no preview
		return m.viewList()
	}

	var s strings.Builder

	// Header spans full width
	header := m.renderHeader()
	s.WriteString(header)
	s.WriteString("\n")

	// Search bar spans full width
	searchBar := m.renderSearchBar()
	s.WriteString(searchBar)
	s.WriteString("\n\n")

	// Render list and preview side by side
	listContent := m.renderNotesListForSplit(listWidth)
	previewContent := m.renderPreview(previewWidth)

	// Split them line by line
	listLines := strings.Split(listContent, "\n")
	previewLines := strings.Split(previewContent, "\n")

	maxLines := len(listLines)
	if len(previewLines) > maxLines {
		maxLines = len(previewLines)
	}

	for i := 0; i < maxLines; i++ {
		var listLine, previewLine string

		if i < len(listLines) {
			listLine = listLines[i]
		}
		if i < len(previewLines) {
			previewLine = previewLines[i]
		}

		// Pad list line to listWidth
		listLine = padRight(listLine, listWidth)

		s.WriteString(listLine)
		s.WriteString("  ") // Separator
		s.WriteString(previewLine)
		s.WriteString("\n")
	}

	// Footer
	footer := m.renderFooter()
	s.WriteString("\n")
	s.WriteString(footer)

	return s.String()
}

func (m dashboardModel) viewHelp() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorPrimary).
		Padding(1, 2)

	sectionStyle := lipgloss.NewStyle().
		Foreground(ui.ColorAccent).
		Bold(true).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(ui.ColorSuccess).
		Bold(true).
		Width(12)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.ColorDefault)

	s.WriteString(titleStyle.Render("LX Dashboard - Keyboard Shortcuts"))
	s.WriteString("\n\n")

	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "Navigation",
			keys: []struct{ key, desc string }{
				{"↑ / k", "Move cursor up"},
				{"↓ / j", "Move cursor down"},
				{"g", "Jump to top"},
				{"G", "Jump to bottom"},
			},
		},
		{
			title: "Actions",
			keys: []struct{ key, desc string }{
				{"Enter / o", "Open note (PDF)"},
				{"e", "Edit note source (.tex)"},
				{"b", "Build note (compile LaTeX)"},
				{"d", "Delete note (with confirmation)"},
				{"n", "Create new note"},
			},
		},
		{
			title: "Views & Search",
			keys: []struct{ key, desc string }{
				{"/", "Start search (type to filter, arrow keys to navigate)"},
				{"Esc", "Exit search / Cancel"},
				{"v", "Toggle graph view"},
				{"?", "Show this help"},
			},
		},
		{
			title: "Preview",
			keys: []struct{ key, desc string }{
				{"PgUp/PgDn", "Scroll preview pane"},
			},
		},
		{
			title: "General",
			keys: []struct{ key, desc string }{
				{"q", "Quit dashboard"},
				{"Ctrl+C", "Force quit"},
			},
		},
	}

	for _, section := range sections {
		s.WriteString(sectionStyle.Render(section.title))
		s.WriteString("\n")
		for _, binding := range section.keys {
			s.WriteString("  ")
			s.WriteString(keyStyle.Render(binding.key))
			s.WriteString(descStyle.Render(binding.desc))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(ui.StyleMuted.Render("  Press ESC or ? to return to dashboard"))
	s.WriteString("\n")

	return s.String()
}

func (m dashboardModel) viewConfirmDelete() string {
	if m.deleteTarget == nil {
		return ""
	}

	var s strings.Builder

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorWarning).
		Padding(1, 2).
		Width(60).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorWarning).
		Bold(true)

	noteStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)

	promptStyle := lipgloss.NewStyle().
		Foreground(ui.ColorDefault).
		MarginTop(1)

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s",
		titleStyle.Render("⚠️  Delete Note?"),
		noteStyle.Render(m.deleteTarget.Title),
		ui.StyleMuted.Render(m.deleteTarget.Slug),
		promptStyle.Render("Press 'y' to confirm, 'n' or ESC to cancel"),
	)

	box := boxStyle.Render(content)

	// Center the box vertically
	verticalPadding := (m.height - lipgloss.Height(box)) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	for i := 0; i < verticalPadding; i++ {
		s.WriteString("\n")
	}

	// Center horizontally
	s.WriteString(lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, box))

	return s.String()
}

func (m dashboardModel) viewGraph() string {
	if m.graphData == nil {
		return "\n  Loading graph...\n"
	}

	var s strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorPrimary).
		Padding(0, 1)

	currentNode := m.getCurrentGraphNode()
	currentTitle := m.getNodeTitle(currentNode)

	s.WriteString("\n")
	s.WriteString(titleStyle.Render("🔗 Graph View"))
	s.WriteString(" ")
	s.WriteString(ui.StyleMuted.Render(fmt.Sprintf("(%d nodes)", len(m.graphData.Nodes))))
	s.WriteString("\n\n")

	// Current node
	s.WriteString(ui.StyleBold.Render("  Current: "))
	s.WriteString(ui.StylePrimary.Render(currentTitle))
	s.WriteString(ui.StyleMuted.Render(fmt.Sprintf(" (%s)", currentNode)))
	s.WriteString("\n\n")

	// Breadcrumb
	if len(m.graphHistory) > 0 {
		prevNode := m.graphHistory[len(m.graphHistory)-1]
		prevTitle := m.getNodeTitle(prevNode)
		s.WriteString(ui.StyleMuted.Render(fmt.Sprintf("  ← Back to: %s (h)", prevTitle)))
		s.WriteString("\n\n")
	}

	// Connections
	neighbors := m.getGraphNeighbors()
	s.WriteString(ui.StyleBold.Render(fmt.Sprintf("  Connections (%d)", len(neighbors))))
	s.WriteString("\n\n")

	if len(neighbors) == 0 {
		s.WriteString(ui.StyleMuted.Render("    (No connections)"))
	} else {
		for i, neighbor := range neighbors {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(ui.ColorDefault)
			if m.graphCursor == i {
				cursor = ui.StyleAccent.Render("  → ")
				style = ui.StyleSuccess.Copy().Bold(true)
			}

			title := m.getNodeTitle(neighbor)
			s.WriteString(cursor)
			s.WriteString(style.Render(title))
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n\n")
	s.WriteString(ui.StyleMuted.Render("  [↑↓] Navigate  [Enter] Explore  [h] Back  [v/Esc] Exit  [q] Quit"))
	s.WriteString("\n")

	return s.String()
}

func (m dashboardModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Padding(0, 1)

	statsStyle := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Align(lipgloss.Right)

	vaultPath := appVault.NotesPath
	if home, err := os.UserHomeDir(); err == nil {
		vaultPath = strings.Replace(vaultPath, home, "~", 1)
	}

	title := titleStyle.Render("📚 LX Notes Dashboard")
	stats := statsStyle.Render(fmt.Sprintf("%d notes  %s", len(m.filteredNotes), vaultPath))

	// Create a two-column layout
	titleWidth := lipgloss.Width(title)
	statsWidth := lipgloss.Width(stats)
	spacer := m.width - titleWidth - statsWidth

	if spacer < 0 {
		spacer = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		strings.Repeat(" ", spacer),
		stats,
	)
}

func (m dashboardModel) renderSearchBar() string {
	borderColor := ui.ColorMuted
	if m.mode == modeSearch {
		borderColor = ui.ColorPrimary
	}

	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(m.width - 4)

	prompt := "🔍 "
	if m.mode == modeSearch {
		prompt = ui.StylePrimary.Render("🔍 ")
	} else {
		prompt = ui.StyleMuted.Render("🔍 ")
	}

	content := prompt + m.searchInput.View()
	if m.mode != modeSearch && m.searchInput.Value() == "" {
		content = prompt + ui.StyleMuted.Render("Press / to search...")
	}

	return searchStyle.Render(content)
}

func (m dashboardModel) renderNotesList() string {
	var s strings.Builder

	if len(m.filteredNotes) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			Italic(true).
			Padding(2, 4)

		if m.searchInput.Value() != "" {
			s.WriteString(emptyStyle.Render("No notes match your search."))
		} else {
			s.WriteString(emptyStyle.Render("No notes found. Press 'n' to create your first note!"))
		}
		return s.String()
	}

	// Calculate viewport
	listHeight := m.height - 10 // Reserve space for header, search, footer
	if listHeight < 3 {
		listHeight = 3
	}

	// Render visible notes
	start := m.offset
	end := m.offset + listHeight
	if end > len(m.filteredNotes) {
		end = len(m.filteredNotes)
	}

	for i := start; i < end; i++ {
		note := m.filteredNotes[i]
		s.WriteString(m.renderNoteItem(note, i == m.cursor))
	}

	return s.String()
}

func (m dashboardModel) renderNoteItem(note domain.NoteHeader, selected bool) string {
	// Styles
	var cursor string
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorDefault)
	metaStyle := ui.StyleMuted

	if selected {
		cursor = ui.StylePrimary.Render("▶ ")
		titleStyle = ui.StylePrimary.Copy().Bold(true)
	} else {
		cursor = "  "
	}

	// Truncate title if too long
	maxTitleLen := m.width - 50
	if maxTitleLen < 20 {
		maxTitleLen = 20
	}
	title := note.Title
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	// Format tags
	tags := ""
	if len(note.Tags) > 0 {
		tagBadges := make([]string, 0, len(note.Tags))
		for _, tag := range note.Tags {
			badge := ui.StyleAccent.Render("[" + tag + "]")
			tagBadges = append(tagBadges, badge)
		}
		tags = strings.Join(tagBadges, " ")
	}

	// Format date
	date := m.formatRelativeTime(note.Date)

	// Build line
	line := fmt.Sprintf("%s%-*s  %s  %s",
		cursor,
		maxTitleLen,
		titleStyle.Render(title),
		metaStyle.Render(date),
		tags,
	)

	return line + "\n"
}

func (m dashboardModel) renderFooter() string {
	// Status message
	var statusLine string
	if m.message != "" && time.Now().Before(m.messageExpiry) {
		statusLine = m.messageStyle.Render(m.message)
	} else {
		statusLine = ui.StyleMuted.Render("Ready")
	}

	// Help hint
	helpHint := ui.StyleMuted.Render("[↑↓/jk] Navigate  [Enter/o] Open  [PgUp/PgDn] Scroll Preview  [/] Search  [?] Help  [q] Quit")

	// Combine
	footerStyle := lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(0, 1)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		statusLine,
		helpHint,
	)

	return footerStyle.Render(content)
}

func (m dashboardModel) renderNotesListForSplit(width int) string {
	var s strings.Builder

	if len(m.filteredNotes) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			Italic(true).
			Padding(2, 2).
			Width(width)

		if m.searchInput.Value() != "" {
			s.WriteString(emptyStyle.Render("No notes match your search."))
		} else {
			s.WriteString(emptyStyle.Render("No notes found."))
		}
		return s.String()
	}

	// Calculate viewport
	listHeight := m.height - 10
	if listHeight < 3 {
		listHeight = 3
	}

	// Render visible notes
	start := m.offset
	end := m.offset + listHeight
	if end > len(m.filteredNotes) {
		end = len(m.filteredNotes)
	}

	for i := start; i < end; i++ {
		note := m.filteredNotes[i]
		line := m.renderNoteItemCompact(note, i == m.cursor, width)
		s.WriteString(line)
	}

	return s.String()
}

func (m dashboardModel) renderNoteItemCompact(note domain.NoteHeader, selected bool, width int) string {
	var cursor string
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorDefault)

	if selected {
		cursor = ui.StylePrimary.Render("▶ ")
		titleStyle = ui.StylePrimary.Copy().Bold(true)
	} else {
		cursor = "  "
	}

	// Truncate title to fit width
	maxTitleLen := width - 15 // Reserve space for cursor and date
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}

	title := note.Title
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	// Format date (compact)
	date := m.formatRelativeTime(note.Date)
	dateStyle := ui.StyleMuted

	line := fmt.Sprintf("%s%-*s %s",
		cursor,
		maxTitleLen,
		titleStyle.Render(title),
		dateStyle.Render(date),
	)

	return padRight(line, width) + "\n"
}

func (m dashboardModel) renderPreview(width int) string {
	var s strings.Builder

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(width - 2).
		Height(m.height - 12)

	if m.preview.content == "" {
		if len(m.filteredNotes) == 0 {
			return borderStyle.Render(
				lipgloss.NewStyle().
					Foreground(ui.ColorMuted).
					Italic(true).
					Padding(1).
					Render("No note selected"),
			)
		}
		return borderStyle.Render(
			lipgloss.NewStyle().
				Foreground(ui.ColorMuted).
				Italic(true).
				Padding(1).
				Render("Loading preview..."),
		)
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Width(width - 4)

	var note *domain.NoteHeader
	for i := range m.filteredNotes {
		if m.filteredNotes[i].Slug == m.preview.slug {
			note = &m.filteredNotes[i]
			break
		}
	}

	if note != nil {
		s.WriteString(titleStyle.Render(note.Title))
		s.WriteString("\n")

		// Tags
		if len(note.Tags) > 0 {
			tagStyle := lipgloss.NewStyle().Foreground(ui.ColorAccent)
			tags := "Tags: "
			for i, tag := range note.Tags {
				if i > 0 {
					tags += ", "
				}
				tags += tag
			}
			s.WriteString(tagStyle.Render(tags))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Scrollable content with syntax highlighting
	s.WriteString(lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(fmt.Sprintf("↑/↓ or PgUp/PgDn to scroll • %d%%", int(m.preview.viewport.ScrollPercent()*100))))
	s.WriteString("\n")
	s.WriteString(m.preview.viewport.View())

	return borderStyle.Render(s.String())
}

func padRight(s string, width int) string {
	// Strip ANSI codes to get real length
	realLen := lipgloss.Width(s)
	if realLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-realLen)
}

func (m *dashboardModel) adjustViewport() {
	listHeight := m.height - 10
	if listHeight < 3 {
		listHeight = 3
	}

	// Scroll down
	if m.cursor >= m.offset+listHeight {
		m.offset = m.cursor - listHeight + 1
	}

	// Scroll up
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}

func (m *dashboardModel) applySearch() {
	query := strings.TrimSpace(m.searchInput.Value())
	if query == "" {
		m.filteredNotes = m.notes
	} else {
		resp, err := listService.Search(m.ctx, services.SearchRequest{Query: query})
		if err == nil {
			m.filteredNotes = resp.Notes
		}
	}

	// Reset cursor
	if m.cursor >= len(m.filteredNotes) {
		m.cursor = len(m.filteredNotes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustViewport()
}

func (m dashboardModel) formatRelativeTime(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}

	// Normalize both dates to midnight for day-based comparison
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	noteDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	diff := today.Sub(noteDate)
	days := int(diff.Hours() / 24)

	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "1d ago"
	case days < 7:
		return fmt.Sprintf("%dd ago", days)
	case days < 14:
		return "1w ago"
	case days < 30:
		return fmt.Sprintf("%dw ago", days/7)
	case days < 60:
		return "1mo ago"
	case days < 365:
		return fmt.Sprintf("%dmo ago", days/30)
	case days < 730:
		return "1y ago"
	default:
		return fmt.Sprintf("%dy ago", days/365)
	}
}

// Commands

type statusMsg struct {
	message string
	style   lipgloss.Style
}

type clearMessageMsg struct{}

type reloadNotesMsg struct{}

type previewLoadedMsg struct {
	slug    string
	content string
}

func (m dashboardModel) openNote(note domain.NoteHeader) tea.Cmd {
	return func() tea.Msg {
		// PDF path is in cache directory with .pdf extension
		pdfPath := appVault.GetCachePath(strings.TrimSuffix(note.Filename, ".tex") + ".pdf")

		// Check if PDF exists
		if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
			return statusMsg{
				message: fmt.Sprintf("PDF not found. Build the note first (press 'b')"),
				style:   ui.StyleWarning,
			}
		}

		// Open PDF
		var cmd *exec.Cmd
		switch {
		case fileExists("/usr/bin/open"): // macOS
			cmd = exec.Command("open", pdfPath)
		case fileExists("/usr/bin/xdg-open"): // Linux
			cmd = exec.Command("xdg-open", pdfPath)
		default:
			return statusMsg{
				message: "Unsupported platform for opening PDFs",
				style:   ui.StyleError,
			}
		}

		if err := cmd.Start(); err != nil {
			return statusMsg{
				message: fmt.Sprintf("Failed to open PDF: %v", err),
				style:   ui.StyleError,
			}
		}

		return statusMsg{
			message: fmt.Sprintf("Opened: %s", note.Title),
			style:   ui.StyleSuccess,
		}
	}
}

func (m dashboardModel) editNote(note domain.NoteHeader) tea.Cmd {
	return func() tea.Msg {
		notePath := appVault.GetNotePath(note.Filename)

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		c := exec.Command(editor, notePath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		return tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return statusMsg{
					message: fmt.Sprintf("Editor error: %v", err),
					style:   ui.StyleError,
				}
			}
			return reloadNotesMsg{}
		})
	}
}

func (m dashboardModel) buildNote(note domain.NoteHeader) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := buildService.Execute(ctx, services.BuildRequest{Slug: note.Slug})
		if err != nil {
			return statusMsg{
				message: fmt.Sprintf("Build failed: %v", err),
				style:   ui.StyleError,
			}
		}

		return statusMsg{
			message: fmt.Sprintf("✓ Built: %s", note.Title),
			style:   ui.StyleSuccess,
		}
	}
}

func (m dashboardModel) deleteNoteConfirmed(note *domain.NoteHeader) tea.Cmd {
	return func() tea.Msg {
		if note == nil {
			return nil
		}

		// Delete note file
		notePath := appVault.GetNotePath(note.Filename)
		if err := os.Remove(notePath); err != nil {
			return statusMsg{
				message: fmt.Sprintf("Failed to delete: %v", err),
				style:   ui.StyleError,
			}
		}

		// Also delete PDF if exists
		pdfPath := appVault.GetCachePath(strings.TrimSuffix(note.Filename, ".tex") + ".pdf")
		os.Remove(pdfPath) // Ignore error if PDF doesn't exist

		// Reload notes
		go func() {
			// Rebuild index in background
			indexerService.Execute(context.Background(), services.ReindexRequest{})
		}()

		// Return success and reload
		return tea.Sequence(
			func() tea.Msg {
				return statusMsg{
					message: fmt.Sprintf("✓ Deleted: %s", note.Title),
					style:   ui.StyleSuccess,
				}
			},
			func() tea.Msg {
				return reloadNotesMsg{}
			},
		)()
	}
}

func (m dashboardModel) createNote() tea.Cmd {
	return func() tea.Msg {
		// This will exit the TUI and prompt for note title
		// For now, we'll show a message
		return statusMsg{
			message: "Note creation from dashboard not yet implemented. Use 'lx new \"title\"'",
			style:   ui.StyleWarning,
		}
	}
}

func (m dashboardModel) loadGraph() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := graphService.GetGraph(ctx, false)
		if err != nil {
			return statusMsg{
				message: fmt.Sprintf("Failed to load graph: %v", err),
				style:   ui.StyleError,
			}
		}

		// Store graph data in model
		return loadGraphMsg{data: &data}
	}
}

func (m dashboardModel) loadPreview(note domain.NoteHeader) tea.Cmd {
	return func() tea.Msg {
		// Read note content
		notePath := appVault.GetNotePath(note.Filename)
		data, err := os.ReadFile(notePath)
		if err != nil {
			return previewLoadedMsg{
				slug:    note.Slug,
				content: fmt.Sprintf("Error loading preview: %v", err),
			}
		}

		content := string(data)

		// Apply syntax highlighting for LaTeX
		highlighted := highlightLatex(content)

		return previewLoadedMsg{
			slug:    note.Slug,
			content: highlighted,
		}
	}
}

// highlightLatex applies syntax highlighting to LaTeX content
func highlightLatex(content string) string {
	lexer := lexers.Get("tex")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.TTY16m

	var buf strings.Builder
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return content
	}

	return buf.String()
}

type loadGraphMsg struct {
	data *services.GraphData
}

func (m *dashboardModel) setGraphNode(nodeID string) {
	// Find current node in graph
	for _, node := range m.graphData.Nodes {
		if node.ID == nodeID {
			m.graphCursor = 0
			return
		}
	}
}

func (m dashboardModel) getCurrentGraphNode() string {
	if m.graphData == nil || len(m.graphData.Nodes) == 0 {
		return ""
	}
	// Use first note if no history
	if len(m.graphHistory) == 0 && len(m.filteredNotes) > 0 {
		return m.filteredNotes[0].Slug
	}
	if len(m.graphHistory) > 0 {
		return m.graphHistory[len(m.graphHistory)-1]
	}
	return m.graphData.Nodes[0].ID
}

func (m dashboardModel) getGraphNeighbors() []string {
	if m.graphData == nil {
		return nil
	}

	currentNode := m.getCurrentGraphNode()
	neighbors := make(map[string]bool)

	for _, link := range m.graphData.Links {
		if link.Source == currentNode {
			neighbors[link.Target] = true
		}
		if link.Target == currentNode {
			neighbors[link.Source] = true
		}
	}

	result := make([]string, 0, len(neighbors))
	for n := range neighbors {
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}

func (m dashboardModel) getNodeTitle(nodeID string) string {
	if m.graphData == nil {
		return nodeID
	}

	for _, node := range m.graphData.Nodes {
		if node.ID == nodeID {
			return node.Title
		}
	}
	return nodeID
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
