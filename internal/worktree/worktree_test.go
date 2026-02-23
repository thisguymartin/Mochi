package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository with an initial commit.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@mochi.local")
	run("git", "config", "user.name", "MOCHI Test")
	run("git", "commit", "--allow-empty", "-m", "initial")

	return dir
}

func newTestManager(t *testing.T, repoRoot string) *Manager {
	t.Helper()
	return NewManager(repoRoot, "main", "feature", filepath.Join(repoRoot, ".worktrees"))
}

func TestResolveBranch_NoConflict(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	got := m.resolveBranch("feature/new-task")
	if got != "feature/new-task" {
		t.Errorf("resolveBranch with no conflict = %q; want %q", got, "feature/new-task")
	}
}

func TestResolveBranch_OneCollision(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	cmd := exec.Command("git", "branch", "feature/my-task")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create test branch: %v\n%s", err, out)
	}

	got := m.resolveBranch("feature/my-task")
	if got != "feature/my-task-2" {
		t.Errorf("resolveBranch with one collision = %q; want %q", got, "feature/my-task-2")
	}
}

func TestResolveBranch_TwoCollisions(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	for _, b := range []string{"feature/my-task", "feature/my-task-2"} {
		cmd := exec.Command("git", "branch", b)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to create test branch %q: %v\n%s", b, err, out)
		}
	}

	got := m.resolveBranch("feature/my-task")
	if got != "feature/my-task-3" {
		t.Errorf("resolveBranch with two collisions = %q; want %q", got, "feature/my-task-3")
	}
}

func TestGetEntry_UnknownSlug(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	// Use a manifest in the temp dir to avoid polluting the package directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("cannot chdir to temp repo: %v", err)
	}
	defer os.Chdir(oldWd)

	_, err = m.GetEntry("definitely-nonexistent-slug-xyz")
	if err == nil {
		t.Fatal("expected error for unknown slug, got nil")
	}
}

func TestList_Empty(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List empty = %d entries; want 0", len(entries))
	}
}

func TestList_WithEntries(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	m.Create("alpha")
	m.Create("beta")

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List = %d entries; want 2", len(entries))
	}
	// Should be sorted
	if entries[0].Slug != "alpha" || entries[1].Slug != "beta" {
		t.Errorf("List order = [%s, %s]; want [alpha, beta]", entries[0].Slug, entries[1].Slug)
	}
}

func TestDestroyAll_Empty(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	removed, err := m.DestroyAll()
	if err != nil {
		t.Fatalf("DestroyAll error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("DestroyAll empty = %d; want 0", len(removed))
	}
}

func TestDestroyAll_WithEntries(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	m.Create("task-a")
	m.Create("task-b")

	removed, err := m.DestroyAll()
	if err != nil {
		t.Fatalf("DestroyAll error: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("DestroyAll = %d removed; want 2", len(removed))
	}

	entries, _ := m.List()
	if len(entries) != 0 {
		t.Errorf("After DestroyAll, List = %d; want 0", len(entries))
	}
}

func TestExists_Found(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	m.Create("my-task")

	if !m.Exists("my-task") {
		t.Error("Exists(\"my-task\") = false; want true")
	}
}

func TestExists_NotFound(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	oldWd, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(oldWd)

	if m.Exists("nonexistent") {
		t.Error("Exists(\"nonexistent\") = true; want false")
	}
}

func TestCreate_Collision(t *testing.T) {
	repoRoot := setupTestRepo(t)
	m := newTestManager(t, repoRoot)

	// Switch to temp repo for manifest writes
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("cannot chdir to temp repo: %v", err)
	}
	defer os.Chdir(oldWd)

	slug := "test-task"
	_, err = m.Create(slug)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Second Create with same slug should fail currently, we want it to succeed
	_, err = m.Create(slug)
	if err != nil {
		t.Errorf("Second Create failed: %v", err)
	}
}
