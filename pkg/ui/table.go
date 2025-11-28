package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableColumn represents a column in the table
type TableColumn struct {
	Header string
	Width  int
	Align  string // "left", "right", "center"
}

// Table represents a data table
type Table struct {
	Columns []TableColumn
	Rows    [][]string
}

// NewTable creates a new table with specified columns
func NewTable(columns []TableColumn) *Table {
	return &Table{
		Columns: columns,
		Rows:    [][]string{},
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells []string) {
	t.Rows = append(t.Rows, cells)
}

// Render renders the table as a string
func (t *Table) Render() string {
	if len(t.Columns) == 0 {
		return ""
	}

	var builder strings.Builder

	// Calculate actual column widths based on content
	colWidths := make([]int, len(t.Columns))
	for i, col := range t.Columns {
		colWidths[i] = len(col.Header)
	}

	// Check row content widths
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Apply minimum widths from column specs
	for i, col := range t.Columns {
		if col.Width > colWidths[i] {
			colWidths[i] = col.Width
		}
	}

	// Render header
	headerParts := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		headerParts[i] = padString(col.Header, colWidths[i], "left")
	}
	headerLine := StyleTableHeader.Render(strings.Join(headerParts, "  "))
	builder.WriteString(headerLine)
	builder.WriteString("\n")

	// Render separator
	separatorParts := make([]string, len(t.Columns))
	for i := range t.Columns {
		separatorParts[i] = strings.Repeat("─", colWidths[i])
	}
	separator := StyleTableBorder.Render(strings.Join(separatorParts, "  "))
	builder.WriteString(separator)
	builder.WriteString("\n")

	// Render rows
	for idx, row := range t.Rows {
		rowParts := make([]string, len(t.Columns))
		for i, cell := range row {
			if i < len(t.Columns) {
				align := t.Columns[i].Align
				if align == "" {
					align = "left"
				}
				rowParts[i] = padString(cell, colWidths[i], align)
			}
		}

		// Alternate row styles
		var rowStyle lipgloss.Style
		if idx%2 == 0 {
			rowStyle = StyleTableRow
		} else {
			rowStyle = StyleTableRowAlt
		}

		rowLine := rowStyle.Render(strings.Join(rowParts, "  "))
		builder.WriteString(rowLine)
		builder.WriteString("\n")
	}

	return builder.String()
}

// padString pads a string to the specified width with alignment
func padString(s string, width int, align string) string {
	if len(s) >= width {
		return s
	}

	padding := width - len(s)

	switch align {
	case "right":
		return strings.Repeat(" ", padding) + s
	case "center":
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
	default: // "left"
		return s + strings.Repeat(" ", padding)
	}
}

// RenderSimpleList renders a simple bulleted list
func RenderSimpleList(items []string) string {
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString(StyleInfo.Render("  • "))
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	return builder.String()
}

// RenderKeyValue renders a key-value pair
func RenderKeyValue(key, value string) string {
	return fmt.Sprintf("%s: %s",
		StyleAccent.Render(key),
		value,
	)
}
