package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
const (
	ColorPrimary = lipgloss.Color("#FF6B9D") // rose pink
	ColorAccent  = lipgloss.Color("#C8B8F5") // lavender
	ColorMuted   = lipgloss.Color("#6C7086") // subdued gray
)

var (
	BannerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Italic(true)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 4)
)
