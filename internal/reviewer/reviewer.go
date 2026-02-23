package reviewer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Options configures a single reviewer invocation.
type Options struct {
	WorktreePath string
	Task         string
	Model        string
	WorkerOutput string
	Iteration    int
	MaxIter      int
	Timeout      int
	Verbose      bool
	LogDir       string
}

// Decision represents the reviewer's verdict.
type Decision struct {
	Done     bool
	Feedback string
	Raw      string
}

const reviewPromptTmpl = `You are a code reviewer evaluating the output of an AI coding agent.

Task the worker was asked to complete:
{{.Task}}

Worker's output (iteration {{.Iteration}} of {{.MaxIter}}):
{{.WorkerOutput}}

Your job:
1. Evaluate whether the task has been completed correctly and completely.
2. Respond with EXACTLY one of:
   - The single word: DONE
   - A line starting with: RETRY: <your feedback>

Rules:
- Reply DONE if the task is fully and correctly completed.
- Reply RETRY: <feedback> if there are issues that must be fixed.
- Be specific in your feedback so the worker can address it.
- Do not include any other text before or after your verdict.

Your verdict:`

type reviewPromptData struct {
	Task         string
	WorkerOutput string
	Iteration    int
	MaxIter      int
}

// Review invokes the reviewer model and returns its decision.
func Review(opts Options) (Decision, error) {
	prompt, err := buildReviewPrompt(opts)
	if err != nil {
		return Decision{}, fmt.Errorf("reviewer: build prompt: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.Timeout)*time.Second)
	defer cancel()

	cmd := buildCommand(ctx, opts.Model, prompt)
	cmd.Dir = opts.WorktreePath

	var outBuf bytes.Buffer
	if opts.Verbose {
		cmd.Stdout = &multiWriter{&outBuf, os.Stdout}
		cmd.Stderr = &multiWriter{&outBuf, os.Stderr}
	} else {
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
	}

	runErr := cmd.Run()

	raw := outBuf.String()

	if opts.LogDir != "" {
		logPath := filepath.Join(opts.LogDir, fmt.Sprintf("%s-reviewer-iter%d.log", slugify(opts.Task), opts.Iteration))
		_ = os.WriteFile(logPath, []byte(raw), 0644)
	}

	if ctx.Err() == context.DeadlineExceeded {
		return Decision{Raw: raw}, fmt.Errorf("reviewer timed out after %ds", opts.Timeout)
	}
	if runErr != nil {
		return Decision{Raw: raw}, fmt.Errorf("reviewer exited with error: %w", runErr)
	}

	return parseDecision(raw), nil
}

// parseDecision scans stdout for the first DONE or RETRY: line.
// It is intentionally lenient so minor extra text from the model doesn't break parsing.
func parseDecision(output string) Decision {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if upper == "DONE" {
			return Decision{Done: true, Raw: output}
		}
		if strings.HasPrefix(upper, "RETRY:") {
			feedback := strings.TrimSpace(trimmed[len("RETRY:"):])
			return Decision{Done: false, Feedback: feedback, Raw: output}
		}
	}
	// Default: if no clear signal, treat as RETRY with the raw output as feedback
	return Decision{Done: false, Feedback: strings.TrimSpace(output), Raw: output}
}

func buildReviewPrompt(opts Options) (string, error) {
	tmpl, err := template.New("review").Parse(reviewPromptTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, reviewPromptData{
		Task:         opts.Task,
		WorkerOutput: truncate(opts.WorkerOutput, 4000),
		Iteration:    opts.Iteration,
		MaxIter:      opts.MaxIter,
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func buildCommand(ctx context.Context, model, prompt string) *exec.Cmd {
	if strings.HasPrefix(model, "gemini-") {
		return exec.CommandContext(ctx, "gemini", "--model", model, "-p", prompt)
	}
	return exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", prompt)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...[truncated]"
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// multiWriter writes to two io.Writers.
type multiWriter struct {
	a, b interface{ Write([]byte) (int, error) }
}

func (m *multiWriter) Write(p []byte) (int, error) {
	m.a.Write(p)
	m.b.Write(p)
	return len(p), nil
}
