package output

import (
	"strings"
	"testing"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/memory"
	"github.com/thisguymartin/ai-forge/internal/source"
)

func TestBuildFileContent(t *testing.T) {
	opts := Options{
		Task:         source.Task{Slug: "fix-auth", Title: "Fix Auth", Model: "claude-opus-4-6"},
		WorkerResult: agent.Result{Output: "Fixed the auth bug."},
		Iterations:   2,
	}
	content := buildFileContent(opts)

	checks := []string{"fix-auth", "Fix Auth", "claude-opus-4-6", "Iterations:** 2", "Fixed the auth bug."}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("buildFileContent missing %q", check)
		}
	}
}

func TestBuildResearchReportContent_WithMemory(t *testing.T) {
	opts := Options{
		Task:         source.Task{Slug: "research-api", Title: "Research API", Model: "gemini-2.5-pro"},
		WorkerResult: agent.Result{Output: "API research output"},
		MemCtx: memory.Context{
			Progress: "Step 1 done",
			Memory:   "Important findings",
			Agents:   "Agent learned X",
		},
		Iterations: 3,
	}
	content := buildResearchReportContent(opts)

	checks := []string{
		"research-api", "Research API", "gemini-2.5-pro",
		"Step 1 done", "Important findings", "Agent learned X",
		"API research output",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("buildResearchReportContent with memory missing %q", check)
		}
	}
}

func TestBuildResearchReportContent_WithoutMemory(t *testing.T) {
	opts := Options{
		Task:         source.Task{Slug: "test", Title: "Test", Model: "claude-opus-4-6"},
		WorkerResult: agent.Result{Output: "output"},
		MemCtx:       memory.Context{},
		Iterations:   1,
	}
	content := buildResearchReportContent(opts)

	if strings.Contains(content, "Worker Memory") {
		t.Error("should not contain Worker Memory section when memory is empty")
	}
	if strings.Contains(content, "Agent Learnings") {
		t.Error("should not contain Agent Learnings section when agents is empty")
	}
}

func TestValidMode_Known(t *testing.T) {
	modes := []string{"pr", "research-report", "audit", "knowledge-base", "issue", "file"}
	for _, m := range modes {
		if !ValidMode(m) {
			t.Errorf("ValidMode(%q) = false; want true", m)
		}
	}
}

func TestValidMode_Unknown(t *testing.T) {
	if ValidMode("unknown-mode") {
		t.Error("ValidMode(\"unknown-mode\") = true; want false")
	}
}

func TestHandle_UnknownMode(t *testing.T) {
	err := Handle(Options{Mode: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
	if !strings.Contains(err.Error(), "unknown output mode") {
		t.Errorf("error = %q; want it to mention 'unknown output mode'", err)
	}
}
