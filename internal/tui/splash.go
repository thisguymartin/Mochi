package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const mochiLogo = `███╗   ███╗ ██████╗  ██████╗██╗  ██╗██╗
████╗ ████║██╔═══██╗██╔════╝██║  ██║██║
██╔████╔██║██║   ██║██║     ███████║██║
██║╚██╔╝██║██║   ██║██║     ██╔══██║██║
██║ ╚═╝ ██║╚██████╔╝╚██████╗██║  ██║██║
╚═╝     ╚═╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝╚═╝`

type tickMsg struct{}

type splashModel struct {
	spinner  spinner.Model
	quitting bool
}

func newSplashModel() splashModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)
	return splashModel{spinner: s}
}

func (m splashModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg{} }),
	)
}

func (m splashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m splashModel) View() string {
	if m.quitting {
		return ""
	}

	logo := BannerStyle.Render(mochiLogo)
	subtitle := SubtitleStyle.
		Align(lipgloss.Center).
		Width(lipgloss.Width(mochiLogo)).
		Render("Multi-Task AI Coding Orchestrator")
	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Render(m.spinner.View() + " Starting up…")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		subtitle,
		"",
		status,
	)

	return BoxStyle.Render(content) + "\n"
}

// RunSplash runs the Bubble Tea splash screen and blocks until it exits.
// It is a no-op when stdout is not a TTY (piped output / CI).
func RunSplash() {
	if !term.IsTerminal(os.Stdout.Fd()) {
		return
	}
	p := tea.NewProgram(newSplashModel())
	_, _ = p.Run()
}

// mochiIconSmall is a compact pixel-art "M" used in the info panel.
const mochiIconSmall = `█▄ ▄█
█ ▀ █
█   █
█   █`

// PrintInfo renders a static, single-shot info panel (no animation).
// Modelled after the Claude Code startup display.
func PrintInfo(version, model, dir string) {
	icon := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Render(mochiIconSmall)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	mutedStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	title := titleStyle.Render("MOCHI") + " " + mutedStyle.Render("v"+version)
	sub := mutedStyle.Render(model + " · Multi-Task AI Coding Orchestrator")
	cwd := mutedStyle.Render(shortenHome(dir))

	info := lipgloss.NewStyle().
		PaddingLeft(2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, sub, cwd))

	content := lipgloss.JoinHorizontal(lipgloss.Center, icon, info)
	fmt.Println(BoxStyle.Render(content))
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + filepath.ToSlash(path[len(home):])
	}
	return path
}
