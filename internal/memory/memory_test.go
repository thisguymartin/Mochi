package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := Load(dir)
	if ctx.Progress != "" || ctx.Memory != "" || ctx.Agents != "" || ctx.Feedback != "" {
		t.Errorf("Load from empty dir should have all empty fields, got %+v", ctx)
	}
}

func TestLoad_WithFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, fileProgress), []byte("progress content"), 0644)
	os.WriteFile(filepath.Join(dir, fileMemory), []byte("memory content"), 0644)
	os.WriteFile(filepath.Join(dir, fileAgents), []byte("agents content"), 0644)
	os.WriteFile(filepath.Join(dir, fileFeedback), []byte("feedback content"), 0644)

	ctx := Load(dir)
	if ctx.Progress != "progress content" {
		t.Errorf("Progress = %q; want %q", ctx.Progress, "progress content")
	}
	if ctx.Memory != "memory content" {
		t.Errorf("Memory = %q; want %q", ctx.Memory, "memory content")
	}
	if ctx.Agents != "agents content" {
		t.Errorf("Agents = %q; want %q", ctx.Agents, "agents content")
	}
	if ctx.Feedback != "feedback content" {
		t.Errorf("Feedback = %q; want %q", ctx.Feedback, "feedback content")
	}
}

func TestWrite_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	data := IterationData{
		Iteration:     1,
		Task:          "test task",
		WorkerOutput:  "output here",
		ReviewerNotes: "looks good",
		Status:        "done",
	}

	if err := Write(dir, data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	for _, name := range []string{fileProgress, fileMemory, fileAgents, fileFeedback} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Write did not create %s", name)
		}
	}
}

func TestHasAny_AllEmpty(t *testing.T) {
	ctx := Context{}
	if ctx.HasAny() {
		t.Error("HasAny() = true for empty context; want false")
	}
}

func TestHasAny_OneSet(t *testing.T) {
	ctx := Context{Progress: "something"}
	if !ctx.HasAny() {
		t.Error("HasAny() = false with Progress set; want true")
	}
}

func TestTruncate_ShortUnchanged(t *testing.T) {
	got := truncate("hello", 100)
	if got != "hello" {
		t.Errorf("truncate short = %q; want %q", got, "hello")
	}
}

func TestTruncate_LongTruncated(t *testing.T) {
	input := "abcdefghij"
	got := truncate(input, 5)
	if got != "abcde\n...[truncated]" {
		t.Errorf("truncate long = %q; want %q", got, "abcde\n...[truncated]")
	}
}
