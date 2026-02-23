package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// Color palette
const (
	ColorPrimary = lipgloss.Color("#FF6B9D") // rose pink
	ColorAccent  = lipgloss.Color("#C8B8F5") // lavender
	ColorMuted   = lipgloss.Color("#6C7086") // subdued gray
	ColorSuccess = lipgloss.Color("#50FA7B") // green
	ColorError   = lipgloss.Color("#FF5555") // red
	ColorWarning = lipgloss.Color("#F1FA8C") // yellow
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

	SuccessStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ColorError)
	WarningStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	MutedStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	BoldStyle    = lipgloss.NewStyle().Bold(true)

	DryRunBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2)

	SummaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			Padding(0, 2)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorMuted)
)

// IsTTY reports whether stdout is connected to a terminal.
func IsTTY() bool {
	return term.IsTerminal(os.Stdout.Fd())
}
