package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette using terminal colors for consistency
	ColorSuccess = lipgloss.AdaptiveColor{Light: "2", Dark: "2"} // Green
	ColorError   = lipgloss.AdaptiveColor{Light: "1", Dark: "1"} // Red
	ColorPrimary = lipgloss.AdaptiveColor{Light: "5", Dark: "5"} // Magenta/Purple
	ColorInfo    = lipgloss.AdaptiveColor{Light: "6", Dark: "6"} // Cyan
	ColorMuted   = lipgloss.AdaptiveColor{Light: "8", Dark: "8"} // Gray
	ColorWarning = lipgloss.AdaptiveColor{Light: "3", Dark: "3"} // Yellow
	ColorAccent  = lipgloss.AdaptiveColor{Light: "4", Dark: "4"} // Blue
	ColorDefault = lipgloss.AdaptiveColor{Light: "7", Dark: "7"} // White

	// Base styles
	StyleSuccess lipgloss.Style
	StyleError   lipgloss.Style
	StylePrimary lipgloss.Style
	StyleInfo    lipgloss.Style
	StyleMuted   lipgloss.Style
	StyleWarning lipgloss.Style
	StyleAccent  lipgloss.Style

	// Component styles
	StyleTitle       lipgloss.Style
	StyleHeader      lipgloss.Style
	StyleSubtle      lipgloss.Style
	StyleBold        lipgloss.Style
	StyleTableHeader lipgloss.Style
	StyleTableRow    lipgloss.Style
	StyleTableRowAlt lipgloss.Style
	StyleTableBorder lipgloss.Style

	// Status icons
	IconSuccess = "✔"
	IconError   = "✘"
	IconRocket  = "🚀"
	IconInfo    = "ℹ"
	IconWarning = "⚠"
	IconNote    = "📝"
	IconBuild   = "🔨"
	IconTag     = "🏷"
)

func init() {
	// Initialize with default (auto) theme
	SetTheme("auto")
}

// SetTheme applies the specified color theme ("auto", "dark", "light")
func SetTheme(theme string) {
	switch theme {
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	default:
		// Auto: lipgloss detects automatically
	}

	// Re-initialize styles
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleError = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	StyleInfo = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StyleAccent = lipgloss.NewStyle().Foreground(ColorAccent)

	StyleTitle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Underline(true)
	StyleHeader = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	StyleSubtle = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
	StyleBold = lipgloss.NewStyle().Bold(true)

	StyleTableHeader = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Align(lipgloss.Left)
	StyleTableRow = lipgloss.NewStyle().Foreground(ColorDefault)
	StyleTableRowAlt = lipgloss.NewStyle().Foreground(ColorDefault).Faint(true)
	StyleTableBorder = lipgloss.NewStyle().Foreground(ColorMuted)
}

// FormatSuccess returns a success message with icon
func FormatSuccess(msg string) string {
	return StyleSuccess.Render(IconSuccess + " " + msg)
}

// FormatError returns an error message with icon
func FormatError(msg string) string {
	return StyleError.Render(IconError + " " + msg)
}

// FormatInfo returns an info message with icon
func FormatInfo(msg string) string {
	return StyleInfo.Render(IconInfo + " " + msg)
}

// FormatWarning returns a warning message with icon
func FormatWarning(msg string) string {
	return StyleWarning.Render(IconWarning + " " + msg)
}

// FormatRocket returns a rocket message (for exciting actions)
func FormatRocket(msg string) string {
	return StylePrimary.Render(IconRocket + " " + msg)
}

// FormatTitle returns a formatted title
func FormatTitle(title string) string {
	return StyleTitle.Render(title)
}

// FormatMuted returns muted/subtle text
func FormatMuted(text string) string {
	return StyleMuted.Render(text)
}

// FormatBold returns bold text
func FormatBold(text string) string {
	return StyleBold.Render(text)
}
