package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileProgress = "PROGRESS.md"
	fileMemory   = "MEMORY.md"
	fileAgents   = "AGENTS.md"
	fileFeedback = "FEEDBACK.md"
)

// Context holds the content of all memory files for a given worktree iteration.
type Context struct {
	Progress string
	Memory   string
	Agents   string
	Feedback string
}

// HasAny returns true if at least one memory file has content.
func (c Context) HasAny() bool {
	return c.Progress != "" || c.Memory != "" || c.Agents != "" || c.Feedback != ""
}

// Load reads all four memory files from worktreePath. Missing files are silently ignored.
func Load(worktreePath string) Context {
	return Context{
		Progress: readFile(filepath.Join(worktreePath, fileProgress)),
		Memory:   readFile(filepath.Join(worktreePath, fileMemory)),
		Agents:   readFile(filepath.Join(worktreePath, fileAgents)),
		Feedback: readFile(filepath.Join(worktreePath, fileFeedback)),
	}
}

// IterationData contains the data to persist after a single iteration.
type IterationData struct {
	Iteration     int
	Task          string
	WorkerOutput  string
	ReviewerNotes string
	Status        string // "in-progress" | "done" | "failed"
}

// Write persists the four memory files into worktreePath based on IterationData.
func Write(worktreePath string, data IterationData) error {
	progress := fmt.Sprintf("# Task Progress\n\n**Task:** %s\n\n**Iteration:** %d\n\n**Status:** %s\n",
		data.Task, data.Iteration, data.Status)

	mem := fmt.Sprintf("# Worker Memory\n\n## Iteration %d Output\n\n%s\n",
		data.Iteration, truncate(data.WorkerOutput, 4000))

	agents := buildAgentsFile(data)

	feedback := ""
	if data.ReviewerNotes != "" {
		feedback = fmt.Sprintf("# Reviewer Feedback\n\n## Iteration %d\n\n%s\n",
			data.Iteration, data.ReviewerNotes)
	}

	files := map[string]string{
		fileProgress: progress,
		fileMemory:   mem,
		fileAgents:   agents,
		fileFeedback: feedback,
	}

	for name, content := range files {
		path := filepath.Join(worktreePath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("memory.Write: cannot write %s: %w", name, err)
		}
	}
	return nil
}

func buildAgentsFile(data IterationData) string {
	var b strings.Builder
	b.WriteString("# Agent Learnings\n\n")
	fmt.Fprintf(&b, "Iteration %d completed with status: %s\n\n", data.Iteration, data.Status)
	if data.ReviewerNotes != "" {
		b.WriteString("## Key Feedback Points\n\n")
		b.WriteString(data.ReviewerNotes)
		b.WriteString("\n\n")
	}
	b.WriteString("## Instructions for Next Iteration\n\n")
	b.WriteString("- Review FEEDBACK.md before starting work\n")
	b.WriteString("- Address all reviewer notes\n")
	b.WriteString("- Build on previous iteration's progress in MEMORY.md\n")
	return b.String()
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...[truncated]"
}
