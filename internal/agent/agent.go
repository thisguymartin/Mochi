package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/thisguymartin/ai-forge/internal/memory"
)

// InvokeOptions configures a single agent invocation.
type InvokeOptions struct {
	WorktreePath  string
	Task          string
	Model         string
	Timeout       int
	LogDir        string
	Verbose       bool
	Iteration     int
	MaxIterations int
	MemoryContext memory.Context
}

// Result captures the outcome of a single agent run.
type Result struct {
	Slug     string
	Success  bool
	Duration time.Duration
	LogPath  string
	Error    error
	Output   string
}

const promptTmpl = `You are an AI coding agent working inside a git worktree.

Worktree path: {{.WorktreePath}}
Current branch: {{.Branch}}

Your task: {{.Task}}
{{- if .HasMemory}}

=== CONTEXT FROM PREVIOUS ITERATIONS ===
{{- if .Feedback}}

## Reviewer Feedback (address this first):
{{.Feedback}}
{{- end}}
{{- if .Progress}}

## Task Progress:
{{.Progress}}
{{- end}}
{{- if .Agents}}

## Agent Learnings:
{{.Agents}}
{{- end}}
=== END CONTEXT ===

IMPORTANT: You are on iteration {{.Iteration}} of {{.MaxIterations}}. Review the feedback above and address all noted issues before proceeding.
{{- end}}

Instructions:
- Focus exclusively on the described task.
- Do not modify files unrelated to this task.
- When finished, commit all changes with a clear, descriptive commit message.
- If the task cannot be completed, create a file named MOCHI_NOTES.md explaining why.

Begin now.`

type promptData struct {
	WorktreePath  string
	Branch        string
	Task          string
	HasMemory     bool
	Feedback      string
	Progress      string
	Agents        string
	Iteration     int
	MaxIterations int
}

// providerFor returns "gemini" if the model name starts with "gemini-",
// otherwise defaults to "claude".
func providerFor(model string) string {
	if strings.HasPrefix(model, "gemini-") {
		return "gemini"
	}
	return "claude"
}

// buildCommand constructs the provider-specific exec.Cmd for non-interactive use.
//
//	claude  → claude --dangerously-skip-permissions -p <prompt>
//	gemini  → gemini --model <model> -p <prompt>
func buildCommand(ctx context.Context, model, prompt string) *exec.Cmd {
	switch providerFor(model) {
	case "gemini":
		return exec.CommandContext(ctx, "gemini", "--model", model, "-p", prompt)
	default:
		return exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", prompt)
	}
}

// Invoke runs the appropriate AI CLI inside the worktree for the given task.
// It writes all output to a log file and returns a Result.
func Invoke(opts InvokeOptions, slug string) Result {
	start := time.Now()

	iteration := opts.Iteration
	if iteration == 0 {
		iteration = 1
	}
	logSuffix := slug
	if opts.MaxIterations > 1 {
		logSuffix = fmt.Sprintf("%s-iter%d", slug, iteration)
	}
	logPath := filepath.Join(opts.LogDir, logSuffix+".log")

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return Result{
			Slug:    slug,
			Success: false,
			Error:   fmt.Errorf("cannot create log file: %w", err),
			LogPath: logPath,
		}
	}
	defer logFile.Close()

	prompt, err := buildPrompt(opts)
	if err != nil {
		return Result{Slug: slug, Success: false, Error: err, LogPath: logPath}
	}

	writeLogHeader(logFile, slug, opts.Model)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.Timeout)*time.Second)
	defer cancel()

	cmd := buildCommand(ctx, opts.Model, prompt)
	cmd.Dir = opts.WorktreePath

	var outBuf bytes.Buffer
	writers := []io.Writer{logFile, &outBuf}
	if opts.Verbose {
		writers = append(writers, os.Stdout)
	}
	mw := io.MultiWriter(writers...)
	cmd.Stdout = mw
	cmd.Stderr = mw

	runErr := cmd.Run()
	duration := time.Since(start)
	writeLogFooter(logFile, slug, opts.Model, duration, runErr)

	output := outBuf.String()

	if ctx.Err() == context.DeadlineExceeded {
		return Result{
			Slug:     slug,
			Success:  false,
			Duration: duration,
			LogPath:  logPath,
			Output:   output,
			Error:    fmt.Errorf("agent timed out after %ds", opts.Timeout),
		}
	}

	if runErr != nil {
		return Result{Slug: slug, Success: false, Duration: duration, LogPath: logPath, Output: output, Error: runErr}
	}

	return Result{Slug: slug, Success: true, Duration: duration, LogPath: logPath, Output: output}
}

func buildPrompt(opts InvokeOptions) (string, error) {
	branch := detectBranch(opts.WorktreePath)

	tmpl, err := template.New("prompt").Parse(promptTmpl)
	if err != nil {
		return "", err
	}

	iteration := opts.Iteration
	if iteration == 0 {
		iteration = 1
	}
	maxIter := opts.MaxIterations
	if maxIter == 0 {
		maxIter = 1
	}

	data := promptData{
		WorktreePath:  opts.WorktreePath,
		Branch:        branch,
		Task:          opts.Task,
		HasMemory:     opts.MemoryContext.HasAny(),
		Feedback:      opts.MemoryContext.Feedback,
		Progress:      opts.MemoryContext.Progress,
		Agents:        opts.MemoryContext.Agents,
		Iteration:     iteration,
		MaxIterations: maxIter,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func detectBranch(worktreePath string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func writeLogHeader(w io.Writer, slug, model string) {
	fmt.Fprintf(w, "[AGENT START] %s | task=%s | model=%s\n",
		time.Now().Format("2006-01-02 15:04:05"), slug, model)
	fmt.Fprintln(w, strings.Repeat("─", 60))
}

func writeLogFooter(w io.Writer, slug, model string, d time.Duration, err error) {
	fmt.Fprintln(w, strings.Repeat("─", 60))
	status := "exit=0"
	if err != nil {
		status = fmt.Sprintf("exit=1 error=%v", err)
	}
	fmt.Fprintf(w, "[AGENT END] %s | task=%s | model=%s | duration=%.0fs | %s\n",
		time.Now().Format("2006-01-02 15:04:05"), slug, model, d.Seconds(), status)
}
