package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

// TestDashboardModelInitialization tests that the dashboard model is initialized correctly
func TestDashboardModelInitialization(t *testing.T) {
	ctx := context.Background()
	notes := []domain.NoteHeader{
		{
			Title:    "Test Note 1",
			Date:     "2024-01-01",
			Tags:     []string{"test", "math"},
			Slug:     "test-note-1",
			Filename: "20240101-test-note-1.tex",
		},
		{
			Title:    "Test Note 2",
			Date:     "2024-01-02",
			Tags:     []string{"physics"},
			Slug:     "test-note-2",
			Filename: "20240102-test-note-2.tex",
		},
	}

	m := newDashboardModel(ctx, notes)

	// Check initial state
	if len(m.notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(m.notes))
	}

	if len(m.filteredNotes) != 2 {
		t.Errorf("Expected 2 filtered notes, got %d", len(m.filteredNotes))
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}

	if m.offset != 0 {
		t.Errorf("Expected offset at 0, got %d", m.offset)
	}

	if m.mode != modeList {
		t.Errorf("Expected mode to be modeList, got %v", m.mode)
	}

	if m.ready {
		t.Error("Expected ready to be false initially")
	}
}

// TestDashboardNavigationUp tests moving cursor up
func TestDashboardNavigationUp(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)
	m.cursor = 2

	// Simulate key press
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 1 {
		t.Errorf("Expected cursor at 1, got %d", m.cursor)
	}
}

// TestDashboardNavigationDown tests moving cursor down
func TestDashboardNavigationDown(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)
	m.cursor = 1

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 2 {
		t.Errorf("Expected cursor at 2, got %d", m.cursor)
	}
}

// TestDashboardNavigationBoundaries tests cursor boundaries
func TestDashboardNavigationBoundaries(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	// Test up boundary (should stay at 0)
	m.cursor = 0
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 0 {
		t.Errorf("Cursor should stay at 0, got %d", m.cursor)
	}

	// Test down boundary (should stay at last item)
	m.cursor = 2 // Last item
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 2 {
		t.Errorf("Cursor should stay at 2, got %d", m.cursor)
	}
}

// TestDashboardJumpToTop tests jumping to top
func TestDashboardJumpToTop(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(10)
	m := newDashboardModel(ctx, notes)
	m.cursor = 5
	m.offset = 3

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}

	if m.offset != 0 {
		t.Errorf("Expected offset at 0, got %d", m.offset)
	}
}

// TestDashboardJumpToBottom tests jumping to bottom
func TestDashboardJumpToBottom(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(10)
	m := newDashboardModel(ctx, notes)
	m.cursor = 2
	m.offset = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 9 {
		t.Errorf("Expected cursor at 9 (last item), got %d", m.cursor)
	}
}

// TestDashboardModeTransitions tests switching between modes
func TestDashboardModeTransitions(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	// Test entering search mode
	if m.mode != modeList {
		t.Errorf("Expected initial mode to be modeList")
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.mode != modeSearch {
		t.Errorf("Expected mode to be modeSearch, got %v", m.mode)
	}

	// Test exiting search mode
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ = m.updateSearch(msg)
	m = updated.(dashboardModel)

	if m.mode != modeList {
		t.Errorf("Expected mode to return to modeList, got %v", m.mode)
	}

	// Test entering help mode
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ = m.updateList(msg)
	m = updated.(dashboardModel)

	if m.mode != modeHelp {
		t.Errorf("Expected mode to be modeHelp, got %v", m.mode)
	}

	// Test exiting help mode
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ = m.updateHelp(msg)
	m = updated.(dashboardModel)

	if m.mode != modeList {
		t.Errorf("Expected mode to return to modeList, got %v", m.mode)
	}
}

// TestDashboardDeleteConfirmation tests delete confirmation flow
func TestDashboardDeleteConfirmation(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)
	m.cursor = 1

	// Trigger delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.mode != modeConfirmDelete {
		t.Errorf("Expected mode to be modeConfirmDelete, got %v", m.mode)
	}

	if m.deleteTarget == nil {
		t.Error("Expected deleteTarget to be set")
	}

	if m.deleteTarget.Title != "Test Note 2" {
		t.Errorf("Expected deleteTarget to be 'Test Note 2', got %s", m.deleteTarget.Title)
	}

	// Test canceling delete
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ = m.updateConfirmDelete(msg)
	m = updated.(dashboardModel)

	if m.mode != modeList {
		t.Errorf("Expected mode to return to modeList")
	}

	if m.deleteTarget != nil {
		t.Error("Expected deleteTarget to be nil after cancel")
	}
}

// TestDashboardSearchFiltering tests search functionality
func TestDashboardSearchFiltering(t *testing.T) {
	ctx := context.Background()
	notes := []domain.NoteHeader{
		{Title: "Calculus Notes", Slug: "calculus-notes", Date: "2024-01-01"},
		{Title: "Physics Homework", Slug: "physics-homework", Date: "2024-01-02"},
		{Title: "Calculus Exercises", Slug: "calculus-exercises", Date: "2024-01-03"},
	}

	m := newDashboardModel(ctx, notes)

	// Note: Since applySearch calls listService.Search which requires actual service,
	// we can't fully test it in unit tests without mocking.
	// We test the state transitions instead.

	// Enter search mode
	m.mode = modeSearch
	m.searchInput.SetValue("calculus")

	// Verify search input is set
	if m.searchInput.Value() != "calculus" {
		t.Errorf("Expected search value to be 'calculus', got %s", m.searchInput.Value())
	}
}

// TestDashboardViewportAdjustment tests viewport scrolling
func TestDashboardViewportAdjustment(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(20)
	m := newDashboardModel(ctx, notes)
	m.height = 20 // Set height for viewport calculation

	// Move cursor down beyond viewport
	m.cursor = 15
	m.adjustViewport()

	// Offset should adjust to keep cursor visible
	listHeight := m.height - 10
	expectedOffset := m.cursor - listHeight + 1
	if expectedOffset < 0 {
		expectedOffset = 0
	}

	if m.offset < 0 {
		t.Errorf("Offset should not be negative, got %d", m.offset)
	}

	// Move cursor back up
	m.cursor = 2
	m.adjustViewport()

	if m.offset > m.cursor {
		t.Errorf("Offset should not be greater than cursor position")
	}
}

// TestDashboardStatusMessage tests status message handling
func TestDashboardStatusMessage(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	msg := statusMsg{
		message: "Test message",
		style:   ui.StyleSuccess,
	}

	updated, _ := m.Update(msg)
	m = updated.(dashboardModel)

	if m.message != "Test message" {
		t.Errorf("Expected message to be 'Test message', got %s", m.message)
	}

	if time.Now().After(m.messageExpiry) {
		t.Error("Message should not be expired immediately")
	}
}

// TestDashboardWindowResize tests window resize handling
func TestDashboardWindowResize(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 40,
	}

	updated, _ := m.Update(msg)
	m = updated.(dashboardModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height to be 40, got %d", m.height)
	}

	if !m.ready {
		t.Error("Expected ready to be true after resize")
	}
}

// TestDashboardRelativeTimeFormatting tests time formatting
func TestDashboardRelativeTimeFormatting(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(1)
	m := newDashboardModel(ctx, notes)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name     string
		date     string
		expected string
	}{
		{
			name:     "today",
			date:     today.Format("2006-01-02"),
			expected: "today",
		},
		{
			name:     "yesterday",
			date:     today.Add(-24 * time.Hour).Format("2006-01-02"),
			expected: "1d ago",
		},
		{
			name:     "week ago",
			date:     today.Add(-7 * 24 * time.Hour).Format("2006-01-02"),
			expected: "1w ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.formatRelativeTime(tt.date)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestDashboardEmptyState tests behavior with no notes
func TestDashboardEmptyState(t *testing.T) {
	ctx := context.Background()
	notes := []domain.NoteHeader{}
	m := newDashboardModel(ctx, notes)

	if len(m.notes) != 0 {
		t.Errorf("Expected 0 notes, got %d", len(m.notes))
	}

	if len(m.filteredNotes) != 0 {
		t.Errorf("Expected 0 filtered notes, got %d", len(m.filteredNotes))
	}

	// Navigation should not crash with empty list
	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, _ = m.updateList(msg)
	// Just checking it doesn't panic
}

// TestDashboardGraphMode tests graph view mode
func TestDashboardGraphMode(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)

	// Set up mock graph data with current node
	m.graphData = &services.GraphData{
		Nodes: []services.GraphNode{
			{ID: "test-note-1", Title: "Test Note 1"},
			{ID: "test-note-2", Title: "Test Note 2"},
			{ID: "test-note-3", Title: "Test Note 3"},
		},
		Links: []services.GraphLink{
			{Source: "test-note-1", Target: "test-note-2"},
			{Source: "test-note-1", Target: "test-note-3"},
		},
	}
	m.mode = modeGraph
	m.graphHistory = []string{"test-note-1"}
	m.graphCursor = 0

	// Test graph navigation - get neighbors first
	neighbors := m.getGraphNeighbors()
	if len(neighbors) < 2 {
		t.Skip("Not enough neighbors for navigation test")
	}

	// Test moving cursor down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.updateGraph(msg)
	m = updated.(dashboardModel)

	if m.graphCursor != 1 {
		t.Errorf("Expected graph cursor at 1, got %d", m.graphCursor)
	}

	// Test exiting graph mode
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ = m.updateGraph(msg)
	m = updated.(dashboardModel)

	if m.mode != modeList {
		t.Errorf("Expected mode to return to modeList")
	}

	if m.graphData != nil {
		t.Error("Expected graphData to be cleared")
	}
}

// TestDashboardRenderingFunctions tests that rendering doesn't crash
func TestDashboardRenderingFunctions(t *testing.T) {
	t.Skip("Skipping rendering tests that require initialized vault")

	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)
	m.width = 100
	m.height = 40
	m.ready = true

	// Test that rendering functions don't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Rendering panicked: %v", r)
		}
	}()

	_ = m.renderSearchBar()
	_ = m.renderNotesList()
	_ = m.renderFooter()
	_ = m.View()
}

// TestDashboardNoteItemRendering tests individual note rendering
func TestDashboardNoteItemRendering(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(1)
	m := newDashboardModel(ctx, notes)
	m.width = 100

	note := notes[0]

	// Render selected
	selectedOutput := m.renderNoteItem(note, true)
	if selectedOutput == "" {
		t.Error("Selected note rendering should not be empty")
	}

	// Render unselected
	unselectedOutput := m.renderNoteItem(note, false)
	if unselectedOutput == "" {
		t.Error("Unselected note rendering should not be empty")
	}

	// They should be different
	if selectedOutput == unselectedOutput {
		t.Error("Selected and unselected renderings should differ")
	}
}

// TestDashboardSearchClearOnEscape tests that search is cleared on escape
func TestDashboardSearchClearOnEscape(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	// Enter search mode and type something
	m.mode = modeSearch
	m.searchInput.SetValue("test query")

	// Press escape
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.updateSearch(msg)
	m = updated.(dashboardModel)

	if m.searchInput.Value() != "" {
		t.Errorf("Expected search to be cleared, got %s", m.searchInput.Value())
	}

	if m.mode != modeList {
		t.Error("Expected to return to list mode")
	}
}

// Helper function to create test notes
func createTestNotes(count int) []domain.NoteHeader {
	notes := make([]domain.NoteHeader, count)
	for i := 0; i < count; i++ {
		notes[i] = domain.NoteHeader{
			Title:    "Test Note " + string(rune('1'+i)),
			Date:     time.Now().Add(-time.Duration(i) * 24 * time.Hour).Format("2006-01-02"),
			Tags:     []string{"test"},
			Slug:     "test-note-" + string(rune('1'+i)),
			Filename: "20240101-test-note-" + string(rune('1'+i)) + ".tex",
		}
	}
	return notes
}

// Benchmark tests
func BenchmarkDashboardRendering(b *testing.B) {
	ctx := context.Background()
	notes := createTestNotes(100)
	m := newDashboardModel(ctx, notes)
	m.width = 100
	m.height = 40
	m.ready = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkDashboardNavigation(b *testing.B) {
	ctx := context.Background()
	notes := createTestNotes(1000)
	m := newDashboardModel(ctx, notes)

	msg := tea.KeyMsg{Type: tea.KeyDown}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updated, _ := m.updateList(msg)
		m = updated.(dashboardModel)
	}
}

// TestDashboardPreviewToggle tests toggling preview on and off
func TestDashboardPreviewToggle(t *testing.T) {
	// Preview is now always enabled, so this test is no longer relevant
	t.Skip("Preview is always enabled now")
}

// TestDashboardSearchModeKeyHandling tests that j/k keys work in search input
func TestDashboardSearchModeKeyHandling(t *testing.T) {
	t.Skip("Skipping: requires initialized listService for applySearch")
	// This test would call applySearch which requires global listService
	// The key behavior is tested in TestDashboardListModeVsSearchModeKeyBindings
}

// TestDashboardSearchModeArrowKeys tests that arrow keys work for navigation in search mode
func TestDashboardSearchModeArrowKeys(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)
	m.mode = modeSearch
	m.searchInput.Focus()
	m.cursor = 1

	// Arrow down should move cursor
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.updateSearch(msg)
	m = updated.(dashboardModel)

	if m.cursor != 2 {
		t.Errorf("Expected cursor at 2 after arrow down, got %d", m.cursor)
	}

	// Arrow up should move cursor back
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updated, _ = m.updateSearch(msg)
	m = updated.(dashboardModel)

	if m.cursor != 1 {
		t.Errorf("Expected cursor at 1 after arrow up, got %d", m.cursor)
	}
}

// TestDashboardSearchModeEnterKey tests that Enter key opens note in search mode
func TestDashboardSearchModeEnterKey(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)
	m.mode = modeSearch
	m.searchInput.Focus()
	m.cursor = 1

	// Press Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.updateSearch(msg)
	m = updated.(dashboardModel)

	// Should exit search mode
	if m.mode != modeList {
		t.Errorf("Expected to exit search mode, still in mode %v", m.mode)
	}

	// Should have a command (open note)
	if cmd == nil {
		t.Error("Expected command to open note")
	}

	// Search input should be blurred
	if m.searchInput.Focused() {
		t.Error("Search input should be blurred after Enter")
	}
}

// TestDashboardPreviewState tests preview state management
func TestDashboardPreviewState(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	// Test initial state
	if m.preview.slug != "" {
		t.Error("Initial preview slug should be empty")
	}
	if m.preview.content != "" {
		t.Error("Initial preview content should be empty")
	}
	// Preview is now always enabled, no need to check

	// Simulate preview loaded
	msg := previewLoadedMsg{
		slug:    "test-note",
		content: "Test content",
	}

	updated, _ := m.Update(msg)
	m = updated.(dashboardModel)

	if m.preview.slug != "test-note" {
		t.Errorf("Expected preview slug 'test-note', got %q", m.preview.slug)
	}

	if m.preview.content != "Test content" {
		t.Errorf("Expected preview content 'Test content', got %q", m.preview.content)
	}
}

// TestDashboardPreviewWithNavigation tests that preview updates when navigating
func TestDashboardPreviewWithNavigation(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)
	m.cursor = 0

	// Note: We can't fully test the preview loading without a vault,
	// but we can verify the command is triggered

	// Move down - should trigger preview load
	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, cmd := m.updateList(msg)

	if cmd == nil {
		t.Error("Expected preview load command when navigating with preview enabled")
	}
}

// TestDashboardPadRight tests the padRight utility function
func TestDashboardPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected int // expected length
	}{
		{"short string", "hello", 10, 10},
		{"exact width", "hello", 5, 5},
		{"longer than width", "hello world", 5, 11}, // Should not truncate
		{"empty string", "", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			actualLen := len(result)

			if actualLen < tt.expected && tt.input != "" && len(tt.input) < tt.width {
				t.Errorf("Expected padded length >= %d, got %d", tt.expected, actualLen)
			}
		})
	}
}

// TestDashboardRenderPreview tests preview rendering
func TestDashboardRenderPreview(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)
	m.width = 100
	m.height = 40

	// Test with no content (should show loading state)
	result := m.renderPreview(50)
	if result == "" {
		t.Error("Preview should render a loading/empty state")
	}

	// Test with content
	m.preview.slug = notes[0].Slug
	m.preview.content = "Test content\nLine 2\nLine 3"
	result = m.renderPreview(50)
	if result == "" {
		t.Error("renderPreview should return content when available")
	}
}

// TestDashboardPreviewLoadedMessage tests preview message handling
func TestDashboardPreviewLoadedMessage(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)

	msg := previewLoadedMsg{
		slug:    "test-slug",
		content: "Test preview content",
	}

	updated, _ := m.Update(msg)
	m = updated.(dashboardModel)

	if m.preview.slug != "test-slug" {
		t.Errorf("Expected preview slug 'test-slug', got %q", m.preview.slug)
	}

	if m.preview.content != "Test preview content" {
		t.Errorf("Expected preview content 'Test preview content', got %q", m.preview.content)
	}
}

// TestDashboardListModeVsSearchModeKeyBindings tests key binding differences
func TestDashboardListModeVsSearchModeKeyBindings(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(5)
	m := newDashboardModel(ctx, notes)

	// In list mode, j/k should move cursor
	m.mode = modeList
	m.cursor = 2

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ := m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 3 {
		t.Errorf("In list mode, 'j' should move cursor down, expected 3 got %d", m.cursor)
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ = m.updateList(msg)
	m = updated.(dashboardModel)

	if m.cursor != 2 {
		t.Errorf("In list mode, 'k' should move cursor up, expected 2 got %d", m.cursor)
	}

	// In search mode, j/k should be typed into search input
	// We verify the key binding logic without triggering applySearch
	m.mode = modeSearch
	m.searchInput.Focus()
	m.searchInput.SetValue("") // Clear input

	// Verify that in search mode, the textinput handles regular keys
	// Arrow keys are used for navigation, not j/k
	msg = tea.KeyMsg{Type: tea.KeyDown}
	m.cursor = 1
	updated, _ = m.updateSearch(msg)
	m = updated.(dashboardModel)

	if m.cursor != 2 {
		t.Error("In search mode, arrow down should move cursor")
	}
}

// TestDashboardRenderNotesListForSplit tests compact list rendering for split view
func TestDashboardRenderNotesListForSplit(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(3)
	m := newDashboardModel(ctx, notes)
	m.width = 100
	m.height = 40

	result := m.renderNotesListForSplit(40)
	if result == "" {
		t.Error("renderNotesListForSplit should return content")
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Error("renderNotesListForSplit should return multiple lines")
	}

	// Test with empty notes
	m.filteredNotes = []domain.NoteHeader{}
	result = m.renderNotesListForSplit(40)
	if result == "" {
		t.Error("renderNotesListForSplit should return empty message when no notes")
	}
}

// TestDashboardRenderNoteItemCompact tests compact note item rendering
func TestDashboardRenderNoteItemCompact(t *testing.T) {
	ctx := context.Background()
	notes := createTestNotes(1)
	m := newDashboardModel(ctx, notes)

	note := notes[0]

	// Test selected
	result := m.renderNoteItemCompact(note, true, 40)
	if result == "" {
		t.Error("renderNoteItemCompact should return content for selected note")
	}

	// Test unselected
	result = m.renderNoteItemCompact(note, false, 40)
	if result == "" {
		t.Error("renderNoteItemCompact should return content for unselected note")
	}
}

// TestDashboardViewWithPreview tests the split view rendering
func TestDashboardViewWithPreview(t *testing.T) {
	t.Skip("Skipping: requires initialized appVault for renderHeader")
	// The viewListWithPreview calls renderHeader which accesses appVault.NotesPath
	// This would be tested in integration tests with a real vault
}

// Benchmark preview rendering
func BenchmarkDashboardPreviewRendering(b *testing.B) {
	ctx := context.Background()
	notes := createTestNotes(10)
	m := newDashboardModel(ctx, notes)
	m.width = 100
	m.height = 40
	m.preview.slug = notes[0].Slug
	m.preview.content = strings.Repeat("Test line\n", 50)
	m.preview.viewport.SetContent(m.preview.content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderPreview(60)
	}
}
