package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// ModelOption represents a selectable AI model.
type ModelOption struct {
	ID       string
	Provider string // "claude" or "gemini"
	Desc     string
}

var models = []ModelOption{
	// Claude
	{ID: "claude-sonnet-4-6", Provider: "claude", Desc: "General purpose (default)"},
	{ID: "claude-opus-4-6", Provider: "claude", Desc: "Complex architecture, migrations"},
	{ID: "claude-haiku-4-5", Provider: "claude", Desc: "Tests, docs, simple fixes"},
	// Gemini
	{ID: "gemini-2.5-pro", Provider: "gemini", Desc: "Complex reasoning, large context"},
	{ID: "gemini-2.0-flash", Provider: "gemini", Desc: "Fast, cost-effective general purpose"},
	{ID: "gemini-1.5-pro", Provider: "gemini", Desc: "Long context, multimodal tasks"},
}

type pickerModel struct {
	cursor   int
	selected string
	quitting bool
}

func newPickerModel(current string) pickerModel {
	cursor := 0
	for i, m := range models {
		if m.ID == current {
			cursor = i
			break
		}
	}
	return pickerModel{cursor: cursor}
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(models)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = models[m.cursor].ID
			m.quitting = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	mutedStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	selectedStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	providerStyle := lipgloss.NewStyle().Foreground(ColorAccent)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select a model") + "\n")
	b.WriteString(mutedStyle.Render("Use ↑/↓ or j/k to navigate, Enter to select, q to cancel") + "\n\n")

	lastProvider := ""
	for i, opt := range models {
		// Show provider header
		if opt.Provider != lastProvider {
			lastProvider = opt.Provider
			b.WriteString(providerStyle.Render(fmt.Sprintf("  %s", strings.ToUpper(opt.Provider))) + "\n")
		}

		cursor := "  "
		style := mutedStyle
		if i == m.cursor {
			cursor = selectedStyle.Render("▸ ")
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		}

		id := style.Render(opt.ID)
		desc := mutedStyle.Render(fmt.Sprintf(" — %s", opt.Desc))
		b.WriteString(fmt.Sprintf("  %s%s%s\n", cursor, id, desc))
	}

	return BoxStyle.Render(b.String())
}

// RunModelPicker displays an interactive model selection TUI.
// Returns the selected model ID, or the current model if cancelled.
func RunModelPicker(current string) (string, error) {
	if !term.IsTerminal(os.Stdout.Fd()) {
		return current, nil
	}

	m := newPickerModel(current)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return current, err
	}

	final := result.(pickerModel)
	if final.selected == "" {
		return current, nil
	}
	return final.selected, nil
}
