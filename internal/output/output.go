package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/memory"
	"github.com/thisguymartin/ai-forge/internal/parser"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

// Mode represents the output dispatch mode.
type Mode string

const (
	ModePR            Mode = "pr"
	ModeResearchReport Mode = "research-report"
	ModeAudit         Mode = "audit"
	ModeKnowledgeBase Mode = "knowledge-base"
	ModeIssue         Mode = "issue"
	ModeFile          Mode = "file"
)

// ValidMode returns true if m is a known output mode.
func ValidMode(m string) bool {
	switch Mode(m) {
	case ModePR, ModeResearchReport, ModeAudit, ModeKnowledgeBase, ModeIssue, ModeFile:
		return true
	}
	return false
}

// Options holds everything needed to dispatch post-loop output.
type Options struct {
	Mode         Mode
	Task         parser.Task
	Entry        *worktree.Entry
	WorkerResult agent.Result
	MemCtx       memory.Context
	Iterations   int
	OutputDir    string
	RepoRoot     string
}

// Handle dispatches the appropriate output handler based on Mode.
// ModePR is intentionally not handled here — it's managed by the orchestrator's
// existing PR creation path.
func Handle(opts Options) error {
	switch opts.Mode {
	case ModePR:
		// PR mode is handled by the orchestrator; nothing to do here.
		return nil
	case ModeFile:
		return handleFile(opts)
	case ModeResearchReport:
		return handleResearchReport(opts)
	case ModeAudit:
		// Stub: future implementation
		return nil
	case ModeKnowledgeBase:
		// Stub: future implementation
		return nil
	case ModeIssue:
		// Stub: future implementation
		return nil
	default:
		return fmt.Errorf("unknown output mode %q", opts.Mode)
	}
}

// handleFile writes the worker output as a plain markdown file to OutputDir.
func handleFile(opts Options) error {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("output: cannot create output dir %q: %w", opts.OutputDir, err)
	}

	filename := fmt.Sprintf("%s.md", opts.Task.Slug)
	path := filepath.Join(opts.OutputDir, filename)

	content := buildFileContent(opts)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("output: cannot write file %q: %w", path, err)
	}
	return nil
}

// handleResearchReport writes a structured research report to OutputDir.
func handleResearchReport(opts Options) error {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("output: cannot create output dir %q: %w", opts.OutputDir, err)
	}

	filename := fmt.Sprintf("%s-report.md", opts.Task.Slug)
	path := filepath.Join(opts.OutputDir, filename)

	content := buildResearchReportContent(opts)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("output: cannot write report %q: %w", path, err)
	}
	return nil
}

func buildFileContent(opts Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Task Output: %s\n\n", opts.Task.Slug)
	fmt.Fprintf(&b, "**Task:** %s\n\n", opts.Task.Description)
	fmt.Fprintf(&b, "**Model:** %s\n\n", opts.Task.Model)
	fmt.Fprintf(&b, "**Iterations:** %d\n\n", opts.Iterations)
	fmt.Fprintf(&b, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	b.WriteString("---\n\n")
	b.WriteString("## Output\n\n")
	b.WriteString(opts.WorkerResult.Output)
	b.WriteString("\n")
	return b.String()
}

func buildResearchReportContent(opts Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Research Report: %s\n\n", opts.Task.Slug)
	fmt.Fprintf(&b, "**Task:** %s\n\n", opts.Task.Description)
	fmt.Fprintf(&b, "**Model:** %s\n\n", opts.Task.Model)
	fmt.Fprintf(&b, "**Iterations completed:** %d\n\n", opts.Iterations)
	fmt.Fprintf(&b, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	b.WriteString("---\n\n")

	if opts.MemCtx.Progress != "" {
		b.WriteString("## Progress Summary\n\n")
		b.WriteString(opts.MemCtx.Progress)
		b.WriteString("\n\n")
	}

	b.WriteString("## Final Output\n\n")
	b.WriteString(opts.WorkerResult.Output)
	b.WriteString("\n\n")

	if opts.MemCtx.Memory != "" {
		b.WriteString("## Worker Memory\n\n")
		b.WriteString(opts.MemCtx.Memory)
		b.WriteString("\n\n")
	}

	if opts.MemCtx.Agents != "" {
		b.WriteString("## Agent Learnings\n\n")
		b.WriteString(opts.MemCtx.Agents)
		b.WriteString("\n")
	}

	return b.String()
}
