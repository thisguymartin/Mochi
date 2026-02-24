package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const manifestFile = ".mochi_manifest.json"

// Entry tracks a single git worktree created by MOCHI.
type Entry struct {
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Status string `json:"status"` // pending | running | done | failed
}

// Manager creates and destroys git worktrees for each task.
type Manager struct {
	BaseBranch   string
	BranchPrefix string
	WorktreeDir  string
	RepoRoot     string
	mu           sync.Mutex
}

// NewManager returns a Manager rooted at repoRoot.
func NewManager(repoRoot, baseBranch, branchPrefix, worktreeDir string) *Manager {
	return &Manager{
		RepoRoot:     repoRoot,
		BaseBranch:   baseBranch,
		BranchPrefix: branchPrefix,
		WorktreeDir:  worktreeDir,
	}
}

// Create spins up a new git worktree for the given slug. If the branch name
// already exists it appends a numeric suffix to avoid collision.
func (m *Manager) Create(slug string) (*Entry, error) {
	if err := m.ensureBaseRefExists(); err != nil {
		return nil, err
	}

	path, _ := filepath.Abs(filepath.Join(m.WorktreeDir, slug))

	// 1. If it's already a worktree, reuse it
	if isWorktree(m.RepoRoot, path) {
		branch := getWorktreeBranch(m.RepoRoot, path)
		if branch != "" {
			entry := &Entry{
				Slug:   slug,
				Path:   path,
				Branch: branch,
				Status: "pending",
			}
			if err := m.saveEntry(entry); err != nil {
				return nil, err
			}
			return entry, nil
		}
	}

	// 2. If directory exists but not a worktree, remove it
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("cannot remove existing non-worktree directory at %q: %w", path, err)
		}
	}

	if err := os.MkdirAll(m.WorktreeDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create worktree dir: %w", err)
	}

	// 3. Decide branch name. If it exists, use suffix to avoid collision.
	branch := m.resolveBranch(fmt.Sprintf("%s/%s", m.BranchPrefix, slug))

	cmd := exec.Command("git", "worktree", "add", "-b", branch, path, m.BaseBranch)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree add failed for %q: %w\n%s", slug, err, string(out))
	}

	entry := &Entry{
		Slug:   slug,
		Path:   path,
		Branch: branch,
		Status: "pending",
	}

	if err := m.saveEntry(entry); err != nil {
		return nil, err
	}

	return entry, nil
}

// Prune runs `git worktree prune` to remove stale registrations and then
// drops any manifest entries whose paths no longer exist on disk.
func (m *Manager) Prune() ([]string, error) {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree prune failed: %w\n%s", err, string(out))
	}

	manifest, err := m.loadManifest()
	if err != nil {
		return nil, err
	}

	var pruned []string
	for slug, entry := range manifest {
		if _, statErr := os.Stat(entry.Path); os.IsNotExist(statErr) {
			if err := m.removeEntry(slug); err != nil {
				return pruned, err
			}
			pruned = append(pruned, slug)
		}
	}
	return pruned, nil
}

// Destroy removes the worktree and deletes its branch.
func (m *Manager) Destroy(slug string) error {
	entry, err := m.GetEntry(slug)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "worktree", "remove", "--force", entry.Path)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w\n%s", err, string(out))
	}

	// Best-effort branch deletion — the branch may already be gone
	exec.Command("git", "branch", "-D", entry.Branch).Run()

	return m.removeEntry(slug)
}

// UpdateStatus sets the status field for a tracked worktree.
func (m *Manager) UpdateStatus(slug, status string) error {
	entry, err := m.GetEntry(slug)
	if err != nil {
		return err
	}
	entry.Status = status
	return m.saveEntry(entry)
}

// GetEntry retrieves an entry from the manifest by slug.
func (m *Manager) GetEntry(slug string) (*Entry, error) {
	manifest, err := m.loadManifest()
	if err != nil {
		return nil, err
	}
	e, ok := manifest[slug]
	if !ok {
		return nil, fmt.Errorf("no worktree entry found for slug %q", slug)
	}
	return &e, nil
}

// resolveBranch returns branchName if it doesn't exist yet, otherwise
// appends -2, -3, ... until it finds an unused name.
func (m *Manager) resolveBranch(branch string) string {
	if !branchExists(m.RepoRoot, branch) {
		return branch
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", branch, i)
		if !branchExists(m.RepoRoot, candidate) {
			return candidate
		}
	}
	return branch
}

// ensureBaseRefExists verifies the base branch exists so
// "git worktree add -b ... path <base>" can succeed. If the repo has no commits
// or the given base branch does not exist, returns a helpful error.
func (m *Manager) ensureBaseRefExists() error {
	if refExists(m.RepoRoot, m.BaseBranch) {
		return nil
	}
	return fmt.Errorf("base branch %q does not exist (repo may have no commits yet). Create an initial commit, e.g.: git commit --allow-empty -m \"Initial commit\", or pass an existing branch with --base-branch", m.BaseBranch)
}

func refExists(repoRoot, ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = repoRoot
	err := cmd.Run()
	return err == nil
}

func branchExists(repoRoot, branch string) bool {
	cmd := exec.Command("git", "branch", "--list", branch)
	cmd.Dir = repoRoot
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out)) != ""
}

func isWorktree(repoRoot, path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = realPath
	}

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			gitPath := strings.TrimPrefix(line, "worktree ")
			realGitPath, err := filepath.EvalSymlinks(gitPath)
			if err == nil {
				gitPath = realGitPath
			}
			if gitPath == absPath {
				return true
			}
		}
	}
	return false
}

func getWorktreeBranch(repoRoot, path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = realPath
	}

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			gitPath := strings.TrimPrefix(line, "worktree ")
			realGitPath, err := filepath.EvalSymlinks(gitPath)
			if err == nil {
				gitPath = realGitPath
			}
			if gitPath == absPath {
				found = true
				continue
			}
		}
		if found && strings.HasPrefix(line, "branch ") {
			return strings.TrimPrefix(line, "branch refs/heads/")
		}
		if found && line == "" {
			break
		}
	}
	return ""
}

func (m *Manager) loadManifest() (map[string]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest := make(map[string]Entry)
	data, err := os.ReadFile(manifestFile)
	if os.IsNotExist(err) {
		return manifest, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (m *Manager) saveEntry(entry *Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest := make(map[string]Entry)
	data, err := os.ReadFile(manifestFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
	}

	manifest[entry.Slug] = *entry
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestFile, out, 0644)
}

func (m *Manager) removeEntry(slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest := make(map[string]Entry)
	data, err := os.ReadFile(manifestFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
	}

	delete(manifest, slug)
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestFile, out, 0644)
}
