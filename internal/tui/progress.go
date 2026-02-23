package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// TaskStatus tracks the state of a single task in the progress display.
type TaskStatus struct {
	Slug      string
	Model     string
	State     string // "pending", "running", "done", "failed"
	Duration  time.Duration
	LastLines []string // last N lines of agent output
}

// ProgressTracker manages the progress TUI state, safe for concurrent use.
type ProgressTracker struct {
	mu       sync.Mutex
	tasks    map[string]*TaskStatus
	order    []string // slug order for display
	phase    string
	started  time.Time
	program  *tea.Program
	isTTY    bool
	done     chan struct{}
}

// NewProgressTracker creates a tracker for the given task slugs.
func NewProgressTracker(slugs []string) *ProgressTracker {
	tasks := make(map[string]*TaskStatus, len(slugs))
	for _, slug := range slugs {
		tasks[slug] = &TaskStatus{
			Slug:  slug,
			State: "pending",
		}
	}
	return &ProgressTracker{
		tasks:   tasks,
		order:   slugs,
		phase:   "Initializing",
		started: time.Now(),
		isTTY:   term.IsTerminal(os.Stdout.Fd()),
		done:    make(chan struct{}),
	}
}

// SetPhase updates the current phase label.
func (pt *ProgressTracker) SetPhase(phase string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.phase = phase
	if !pt.isTTY {
		fmt.Printf("[MOCHI] %s\n", phase)
	}
	if pt.program != nil {
		pt.program.Send(refreshMsg{})
	}
}

// UpdateTask updates a task's state.
func (pt *ProgressTracker) UpdateTask(slug, state, model string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if t, ok := pt.tasks[slug]; ok {
		t.State = state
		if model != "" {
			t.Model = model
		}
		if !pt.isTTY {
			fmt.Printf("  %s  %-30s [%s]\n", stateIcon(state), slug, t.Model)
		}
	}
	if pt.program != nil {
		pt.program.Send(refreshMsg{})
	}
}

// SetTaskDuration records the final duration for a task.
func (pt *ProgressTracker) SetTaskDuration(slug string, d time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if t, ok := pt.tasks[slug]; ok {
		t.Duration = d
	}
}

// AppendOutput adds output lines for a task (keeps last 5).
func (pt *ProgressTracker) AppendOutput(slug, line string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if t, ok := pt.tasks[slug]; ok {
		t.LastLines = append(t.LastLines, line)
		if len(t.LastLines) > 5 {
			t.LastLines = t.LastLines[len(t.LastLines)-5:]
		}
	}
	if pt.program != nil {
		pt.program.Send(refreshMsg{})
	}
}

// Start launches the TUI (blocks in a goroutine). No-op if not a TTY.
func (pt *ProgressTracker) Start() {
	if !pt.isTTY {
		return
	}
	go func() {
		m := newProgressModel(pt)
		pt.program = tea.NewProgram(m)
		_, _ = pt.program.Run()
		close(pt.done)
	}()
}

// Stop signals the TUI to quit and waits for it to finish.
func (pt *ProgressTracker) Stop() {
	if pt.program != nil {
		pt.program.Send(quitMsg{})
		<-pt.done
	}
}

// PrintSummary outputs the final summary (works in both TTY and non-TTY).
func (pt *ProgressTracker) PrintSummary() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	succeeded, failed := 0, 0
	for _, slug := range pt.order {
		t := pt.tasks[slug]
		if t.State == "done" {
			succeeded++
		} else if t.State == "failed" {
			failed++
		}
	}

	elapsed := time.Since(pt.started).Truncate(time.Second)
	fmt.Println()
	fmt.Println(styleBold2.Render("─────────────────────────────────────────────────"))
	line := fmt.Sprintf("[MOCHI] Run complete: %d succeeded, %d failed (%s)", succeeded, failed, elapsed)
	if failed == 0 {
		fmt.Println(styleGreen2.Render(line))
	} else {
		fmt.Println(styleRed2.Render(line))
	}
	fmt.Println(styleBold2.Render("─────────────────────────────────────────────────"))
}

// ── Bubble Tea model ───────────────────────────────────────────────────────

type refreshMsg struct{}
type quitMsg struct{}
type tickMsg2 struct{}

var (
	styleGreen2  = SuccessStyle
	styleRed2    = ErrorStyle
	styleYellow2 = WarningStyle
	styleBold2   = BoldStyle
	styleDim     = MutedStyle
)

type progressModel struct {
	tracker  *ProgressTracker
	spinner  spinner.Model
	quitting bool
}

func newProgressModel(pt *ProgressTracker) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)
	return progressModel{tracker: pt, spinner: s}
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg2{} }),
	)
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case quitMsg:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyMsg:
		// Allow q/ctrl+c to quit
		key := msg.(tea.KeyMsg).String()
		if key == "q" || key == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case refreshMsg:
		return m, nil
	case tickMsg2:
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg2{} })
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.quitting {
		return ""
	}

	m.tracker.mu.Lock()
	defer m.tracker.mu.Unlock()

	var b strings.Builder

	// Phase header
	b.WriteString(fmt.Sprintf("%s %s %s\n\n",
		m.spinner.View(),
		styleBold2.Render("[MOCHI]"),
		m.tracker.phase))

	// Task list
	for _, slug := range m.tracker.order {
		t := m.tracker.tasks[slug]
		icon := stateIcon(t.State)
		model := t.Model
		if model == "" {
			model = "…"
		}

		switch t.State {
		case "done":
			b.WriteString(fmt.Sprintf("  %s %-30s [%s] (%.0fs)\n",
				styleGreen2.Render(icon), slug, model, t.Duration.Seconds()))
		case "failed":
			b.WriteString(fmt.Sprintf("  %s %-30s [%s] (%.0fs)\n",
				styleRed2.Render(icon), slug, model, t.Duration.Seconds()))
		case "running":
			b.WriteString(fmt.Sprintf("  %s %-30s [%s]\n",
				styleYellow2.Render(icon), slug, model))
			// Show last output lines
			for _, line := range t.LastLines {
				truncated := line
				if len(truncated) > 80 {
					truncated = truncated[:77] + "..."
				}
				b.WriteString(fmt.Sprintf("      %s\n", styleDim.Render(truncated)))
			}
		default:
			b.WriteString(fmt.Sprintf("  %s %-30s [%s]\n",
				styleDim.Render(icon), slug, model))
		}
	}

	// Elapsed time
	elapsed := time.Since(m.tracker.started).Truncate(time.Second)
	b.WriteString(fmt.Sprintf("\n  %s\n", styleDim.Render(fmt.Sprintf("Elapsed: %s", elapsed))))

	return b.String()
}

func stateIcon(state string) string {
	switch state {
	case "done":
		return "✓"
	case "failed":
		return "✗"
	case "running":
		return "⟳"
	default:
		return "○"
	}
}
