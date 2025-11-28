package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"

	"github.com/gdamore/tcell/v2"
)

// InteractiveGraphView provides a terminal-based interactive graph explorer
type InteractiveGraphView struct {
	index         *domain.Index
	currentSlug   string
	history       []string // Navigation history
	historyIndex  int
	screen        tcell.Screen
	width         int
	height        int
	scrollOffset  int
	selectedIndex int
	mode          string // "note" or "connections"
}

// NewInteractiveGraphView creates a new interactive graph viewer
func NewInteractiveGraphView(index *domain.Index, startSlug string) (*InteractiveGraphView, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := screen.Init(); err != nil {
		return nil, err
	}

	width, height := screen.Size()

	return &InteractiveGraphView{
		index:         index,
		currentSlug:   startSlug,
		history:       []string{startSlug},
		historyIndex:  0,
		screen:        screen,
		width:         width,
		height:        height,
		scrollOffset:  0,
		selectedIndex: 0,
		mode:          "connections",
	}, nil
}

// Run starts the interactive viewer
func (v *InteractiveGraphView) Run() error {
	defer v.screen.Fini()

	v.screen.Clear()
	v.render()

	for {
		ev := v.screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			v.width, v.height = ev.Size()
			v.screen.Sync()
			v.render()

		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'q' {
				return nil
			}

			v.handleKeyPress(ev)
			v.render()
		}
	}
}

// handleKeyPress processes keyboard input
func (v *InteractiveGraphView) handleKeyPress(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyUp, tcell.KeyCtrlP:
		v.moveCursor(-1)
	case tcell.KeyDown, tcell.KeyCtrlN:
		v.moveCursor(1)
	case tcell.KeyEnter:
		v.navigateToSelected()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		v.goBack()
	case tcell.KeyCtrlF:
		v.goForward()
	case tcell.KeyTab:
		v.toggleMode()
	case tcell.KeyHome:
		v.selectedIndex = 0
		v.scrollOffset = 0
	case tcell.KeyEnd:
		items := v.getNavigableItems()
		v.selectedIndex = len(items) - 1
		v.adjustScroll()
	}

	// Vim-style navigation
	switch ev.Rune() {
	case 'j':
		v.moveCursor(1)
	case 'k':
		v.moveCursor(-1)
	case 'h':
		v.goBack()
	case 'l':
		v.navigateToSelected()
	case 'g':
		v.selectedIndex = 0
		v.scrollOffset = 0
	case 'G':
		items := v.getNavigableItems()
		v.selectedIndex = len(items) - 1
		v.adjustScroll()
	}
}

// moveCursor moves the selection cursor
func (v *InteractiveGraphView) moveCursor(delta int) {
	items := v.getNavigableItems()
	if len(items) == 0 {
		return
	}

	v.selectedIndex += delta

	if v.selectedIndex < 0 {
		v.selectedIndex = 0
	}
	if v.selectedIndex >= len(items) {
		v.selectedIndex = len(items) - 1
	}

	v.adjustScroll()
}

// adjustScroll adjusts scroll offset to keep cursor visible
func (v *InteractiveGraphView) adjustScroll() {
	visibleLines := v.height - 8 // Reserve space for header/footer

	if v.selectedIndex < v.scrollOffset {
		v.scrollOffset = v.selectedIndex
	}
	if v.selectedIndex >= v.scrollOffset+visibleLines {
		v.scrollOffset = v.selectedIndex - visibleLines + 1
	}
}

// getNavigableItems returns list of items that can be navigated to
func (v *InteractiveGraphView) getNavigableItems() []string {
	entry, exists := v.index.GetNote(v.currentSlug)
	if !exists {
		return []string{}
	}

	items := []string{}
	items = append(items, entry.Backlinks...)
	items = append(items, entry.OutgoingLinks...)

	// Sort for consistency
	sort.Strings(items)

	return items
}

// navigateToSelected navigates to the selected item
func (v *InteractiveGraphView) navigateToSelected() {
	items := v.getNavigableItems()
	if len(items) == 0 || v.selectedIndex >= len(items) {
		return
	}

	targetSlug := items[v.selectedIndex]
	if !v.index.HasNote(targetSlug) {
		return
	}

	// Add to history
	if v.historyIndex < len(v.history)-1 {
		v.history = v.history[:v.historyIndex+1]
	}
	v.history = append(v.history, targetSlug)
	v.historyIndex++

	// Navigate
	v.currentSlug = targetSlug
	v.selectedIndex = 0
	v.scrollOffset = 0
}

// goBack navigates to previous note in history
func (v *InteractiveGraphView) goBack() {
	if v.historyIndex > 0 {
		v.historyIndex--
		v.currentSlug = v.history[v.historyIndex]
		v.selectedIndex = 0
		v.scrollOffset = 0
	}
}

// goForward navigates to next note in history
func (v *InteractiveGraphView) goForward() {
	if v.historyIndex < len(v.history)-1 {
		v.historyIndex++
		v.currentSlug = v.history[v.historyIndex]
		v.selectedIndex = 0
		v.scrollOffset = 0
	}
}

// toggleMode switches between viewing modes
func (v *InteractiveGraphView) toggleMode() {
	if v.mode == "connections" {
		v.mode = "note"
	} else {
		v.mode = "connections"
	}
}

// render draws the interface
func (v *InteractiveGraphView) render() {
	v.screen.Clear()

	entry, exists := v.index.GetNote(v.currentSlug)
	if !exists {
		v.drawText(0, 0, "Note not found: "+v.currentSlug, tcell.StyleDefault)
		v.screen.Show()
		return
	}

	y := 0

	// Header - Note title
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorPurple)
	v.drawText(0, y, "┌─ "+entry.Title, titleStyle)
	y++
	v.drawText(0, y, "│  "+entry.Date+" │ Tags: "+strings.Join(entry.Tags, ", "), tcell.StyleDefault.Foreground(tcell.ColorGray))
	y++
	v.drawText(0, y, "└─────────────────────────────────────────────────────────────", tcell.StyleDefault.Foreground(tcell.ColorGray))
	y += 2

	// Stats
	totalConnections := len(entry.Backlinks) + len(entry.OutgoingLinks)
	statsText := fmt.Sprintf("Connections: %d | Backlinks: %d | Outgoing: %d",
		totalConnections, len(entry.Backlinks), len(entry.OutgoingLinks))
	v.drawText(0, y, statsText, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	y += 2

	// Backlinks section
	if len(entry.Backlinks) > 0 {
		headerStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.Color51)
		v.drawText(0, y, "← BACKLINKS", headerStyle)
		y++

		backlinks := make([]string, len(entry.Backlinks))
		copy(backlinks, entry.Backlinks)
		sort.Strings(backlinks)

		for i, slug := range backlinks {
			if y-6 < v.scrollOffset {
				continue
			}
			if y-6 >= v.scrollOffset+(v.height-8) {
				break
			}

			itemIndex := i
			style := tcell.StyleDefault
			prefix := "  "

			if itemIndex == v.selectedIndex {
				style = style.Reverse(true)
				prefix = "▶ "
			}

			linkEntry, exists := v.index.GetNote(slug)
			displayText := slug
			if exists {
				displayText = linkEntry.Title
			}

			v.drawText(0, y, prefix+"← "+displayText, style)
			y++
		}
		y++
	}

	// Outgoing links section
	if len(entry.OutgoingLinks) > 0 {
		headerStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGreen)
		v.drawText(0, y, "→ OUTGOING LINKS", headerStyle)
		y++

		outgoing := make([]string, len(entry.OutgoingLinks))
		copy(outgoing, entry.OutgoingLinks)
		sort.Strings(outgoing)

		for i, slug := range outgoing {
			itemIndex := len(entry.Backlinks) + i

			if y-6 < v.scrollOffset {
				continue
			}
			if y-6 >= v.scrollOffset+(v.height-8) {
				break
			}

			style := tcell.StyleDefault
			prefix := "  "

			if itemIndex == v.selectedIndex {
				style = style.Reverse(true)
				prefix = "▶ "
			}

			linkEntry, exists := v.index.GetNote(slug)
			displayText := slug
			if exists {
				displayText = linkEntry.Title
			}

			v.drawText(0, y, prefix+"→ "+displayText, style)
			y++
		}
	}

	// Footer - Help text
	footerY := v.height - 2
	v.drawText(0, footerY, strings.Repeat("─", v.width), tcell.StyleDefault.Foreground(tcell.ColorGray))
	footerY++

	helpText := "↑↓/jk: Navigate │ Enter/l: Open │ Backspace/h: Back │ q/Esc: Quit"
	v.drawText(0, footerY, helpText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
}

// drawText draws text at the specified position
func (v *InteractiveGraphView) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		if x+i >= v.width {
			break
		}
		v.screen.SetContent(x+i, y, r, nil, style)
	}
}
