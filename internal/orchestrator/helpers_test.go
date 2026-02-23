package orchestrator

import (
	"testing"

	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/source"
)

func TestFilterBySlug_Found(t *testing.T) {
	tasks := []source.Task{
		{Slug: "fix-auth"},
		{Slug: "add-tests"},
	}
	got := filterBySlug(tasks, "add-tests")
	if len(got) != 1 || got[0].Slug != "add-tests" {
		t.Errorf("filterBySlug found = %v; want [add-tests]", got)
	}
}

func TestFilterBySlug_NotFound(t *testing.T) {
	tasks := []source.Task{
		{Slug: "fix-auth"},
	}
	got := filterBySlug(tasks, "nonexistent")
	if got != nil {
		t.Errorf("filterBySlug not found = %v; want nil", got)
	}
}

func TestFilterBySlug_MultipleOneMatch(t *testing.T) {
	tasks := []source.Task{
		{Slug: "task-a"},
		{Slug: "task-b"},
		{Slug: "task-c"},
	}
	got := filterBySlug(tasks, "task-b")
	if len(got) != 1 || got[0].Slug != "task-b" {
		t.Errorf("filterBySlug = %v; want [task-b]", got)
	}
}

func TestSlugList_Empty(t *testing.T) {
	got := slugList(nil)
	if got != "" {
		t.Errorf("slugList(nil) = %q; want %q", got, "")
	}
}

func TestSlugList_Single(t *testing.T) {
	tasks := []source.Task{{Slug: "fix-auth"}}
	got := slugList(tasks)
	if got != "fix-auth" {
		t.Errorf("slugList single = %q; want %q", got, "fix-auth")
	}
}

func TestSlugList_Multiple(t *testing.T) {
	tasks := []source.Task{
		{Slug: "task-a"},
		{Slug: "task-b"},
	}
	got := slugList(tasks)
	if got != "task-a, task-b" {
		t.Errorf("slugList multiple = %q; want %q", got, "task-a, task-b")
	}
}

func TestStatusStr_True(t *testing.T) {
	if got := statusStr(true); got != "done" {
		t.Errorf("statusStr(true) = %q; want %q", got, "done")
	}
}

func TestStatusStr_False(t *testing.T) {
	if got := statusStr(false); got != "failed" {
		t.Errorf("statusStr(false) = %q; want %q", got, "failed")
	}
}

func TestLoopEnabled_NoReviewerNoLoop(t *testing.T) {
	cfg := config.Config{ReviewerModel: "", MaxIterations: 1}
	if loopEnabled(cfg) {
		t.Error("loopEnabled should be false with no reviewer and maxIterations=1")
	}
}

func TestLoopEnabled_WithReviewer(t *testing.T) {
	cfg := config.Config{ReviewerModel: "claude-opus-4-6", MaxIterations: 1}
	if !loopEnabled(cfg) {
		t.Error("loopEnabled should be true with reviewer set")
	}
}

func TestLoopEnabled_WithIterations(t *testing.T) {
	cfg := config.Config{ReviewerModel: "", MaxIterations: 3}
	if !loopEnabled(cfg) {
		t.Error("loopEnabled should be true with maxIterations > 1")
	}
}
