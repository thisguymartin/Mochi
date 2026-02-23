package agent

import (
	"strings"
	"testing"

	"github.com/thisguymartin/ai-forge/internal/memory"
)

func TestProviderFor_Gemini(t *testing.T) {
	if got := providerFor("gemini-2.5-pro"); got != "gemini" {
		t.Errorf("providerFor(\"gemini-2.5-pro\") = %q; want %q", got, "gemini")
	}
}

func TestProviderFor_Claude(t *testing.T) {
	if got := providerFor("claude-opus-4-6"); got != "claude" {
		t.Errorf("providerFor(\"claude-opus-4-6\") = %q; want %q", got, "claude")
	}
}

func TestProviderFor_Unknown(t *testing.T) {
	if got := providerFor("gpt-4"); got != "claude" {
		t.Errorf("providerFor(\"gpt-4\") = %q; want %q (defaults to claude)", got, "claude")
	}
}

func TestBuildPrompt_Basic(t *testing.T) {
	opts := InvokeOptions{
		WorktreePath:  "/tmp/worktree",
		Task:          "Fix the auth bug",
		Model:         "claude-opus-4-6",
		Iteration:     1,
		MaxIterations: 1,
	}
	prompt, err := buildPrompt(opts)
	if err != nil {
		t.Fatalf("buildPrompt error: %v", err)
	}
	if !strings.Contains(prompt, "Fix the auth bug") {
		t.Error("prompt should contain the task")
	}
	if !strings.Contains(prompt, "/tmp/worktree") {
		t.Error("prompt should contain the worktree path")
	}
}

func TestBuildPrompt_WithMemory(t *testing.T) {
	opts := InvokeOptions{
		WorktreePath:  "/tmp/worktree",
		Task:          "Fix tests",
		Model:         "claude-opus-4-6",
		Iteration:     2,
		MaxIterations: 3,
		MemoryContext: memory.Context{
			Feedback: "Tests still failing",
			Progress: "Iteration 1 complete",
		},
	}
	prompt, err := buildPrompt(opts)
	if err != nil {
		t.Fatalf("buildPrompt error: %v", err)
	}
	if !strings.Contains(prompt, "Tests still failing") {
		t.Error("prompt should contain feedback")
	}
	if !strings.Contains(prompt, "Iteration 1 complete") {
		t.Error("prompt should contain progress")
	}
	if !strings.Contains(prompt, "iteration 2 of 3") {
		t.Error("prompt should mention iteration count")
	}
}

func TestBuildPrompt_WithoutMemory(t *testing.T) {
	opts := InvokeOptions{
		WorktreePath:  "/tmp/worktree",
		Task:          "Add feature",
		Model:         "claude-opus-4-6",
		Iteration:     1,
		MaxIterations: 1,
		MemoryContext: memory.Context{},
	}
	prompt, err := buildPrompt(opts)
	if err != nil {
		t.Fatalf("buildPrompt error: %v", err)
	}
	if strings.Contains(prompt, "CONTEXT FROM PREVIOUS ITERATIONS") {
		t.Error("prompt should not contain memory section when no memory")
	}
}
