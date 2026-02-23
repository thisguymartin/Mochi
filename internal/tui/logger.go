package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Logger is the shared MOCHI logger.
var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          "MOCHI",
	})

	styles := log.DefaultStyles()
	styles.Prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	styles.Timestamp = lipgloss.NewStyle().Foreground(ColorMuted)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("INFO").
		Foreground(ColorAccent).
		Bold(true)
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("WARN").
		Foreground(ColorWarning).
		Bold(true)
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("ERRO").
		Foreground(ColorError).
		Bold(true)
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("DBUG").
		Foreground(ColorMuted)
	Logger.SetStyles(styles)
}

// SetVerbose sets the logger to debug level when verbose is true.
func SetVerbose(verbose bool) {
	if verbose {
		Logger.SetLevel(log.DebugLevel)
	} else {
		Logger.SetLevel(log.InfoLevel)
	}
}
