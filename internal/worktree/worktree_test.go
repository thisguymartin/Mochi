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
