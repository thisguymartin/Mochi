package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/config"
	gh "github.com/thisguymartin/ai-forge/internal/github"
	"github.com/thisguymartin/ai-forge/internal/parser"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

// Run is the main entry point for a RALPHY execution cycle.
// It orchestrates parsing, worktree creation, agent invocation, PR creation, and cleanup.
func Run(cfg config.Config) error {
	printBanner()

	// ── 1. Resolve task source ─────────────────────────────────────────────
	taskFile, cleanup, err := resolveTaskFile(cfg)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// ── 2. Parse tasks ─────────────────────────────────────────────────────
	tasks, err := parser.ParseFile(taskFile)
	if err != nil {
		return err
	}

	// Apply single-task filter
	if cfg.TaskFilter != "" {
		tasks = filterBySlug(tasks, cfg.TaskFilter)
		if len(tasks) == 0 {
			return fmt.Errorf("no task found with slug %q", cfg.TaskFilter)
		}
	}

	// Apply default model to tasks that don't specify one
	for i := range tasks {
		if tasks[i].Model == "" {
			tasks[i].Model = cfg.Model
		}
	}

	printSection(fmt.Sprintf("Found %d task(s): %s", len(tasks), slugList(tasks)))

	// ── 3. Dry run ─────────────────────────────────────────────────────────
	if cfg.DryRun {
		return printDryRun(tasks, cfg)
	}

	// ── 4. Setup ───────────────────────────────────────────────────────────
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("cannot create log dir %q: %w", cfg.LogDir, err)
	}

	wm := worktree.NewManager(repoRoot, cfg.BaseBranch, cfg.BranchPrefix, cfg.WorktreeDir)

	// ── 5. Create worktrees ────────────────────────────────────────────────
	printSection("Creating worktrees...")
	entries := make([]*worktree.Entry, 0, len(tasks))
	for _, t := range tasks {
		entry, err := wm.Create(t.Slug)
		if err != nil {
			printFail(fmt.Sprintf("%-30s %v", t.Slug, err))
			return err
		}
		entries = append(entries, entry)
		printSuccess(fmt.Sprintf("%-30s (%s)", entry.Path, entry.Branch))
	}

	// ── 6. Invoke agents ───────────────────────────────────────────────────
	printSection("Invoking agents...")
	results := make([]agent.Result, len(tasks))

	if cfg.Sequential {
		for i, t := range tasks {
			printInfo(fmt.Sprintf("⟳  %-28s [%s]", t.Slug, t.Model))
			_ = wm.UpdateStatus(t.Slug, "running")
			results[i] = agent.Invoke(agent.InvokeOptions{
				WorktreePath: entries[i].Path,
				Task:         t.Description,
				Model:        t.Model,
				Timeout:      cfg.Timeout,
				LogDir:       cfg.LogDir,
				Verbose:      cfg.Verbose,
			}, t.Slug)
			_ = wm.UpdateStatus(t.Slug, statusStr(results[i].Success))
			printResult(results[i])
		}
	} else {
		var wg sync.WaitGroup
		for i, t := range tasks {
			wg.Add(1)
			go func(idx int, task parser.Task, entry *worktree.Entry) {
				defer wg.Done()
				printInfo(fmt.Sprintf("⟳  %-28s [%s]", task.Slug, task.Model))
				_ = wm.UpdateStatus(task.Slug, "running")
				results[idx] = agent.Invoke(agent.InvokeOptions{
					WorktreePath: entry.Path,
					Task:         task.Description,
					Model:        task.Model,
					Timeout:      cfg.Timeout,
					LogDir:       cfg.LogDir,
					Verbose:      cfg.Verbose,
				}, task.Slug)
				_ = wm.UpdateStatus(task.Slug, statusStr(results[idx].Success))
				printResult(results[idx])
			}(i, t, entries[i])
		}
		wg.Wait()
	}

	// ── 7. Create PRs ──────────────────────────────────────────────────────
	if cfg.CreatePRs {
		printSection("Creating pull requests...")
		for i, t := range tasks {
			if !results[i].Success {
				printWarn(fmt.Sprintf("Skipping PR for %-24s (agent failed)", t.Slug))
				continue
			}
			if err := gh.PushBranch(repoRoot, entries[i].Branch); err != nil {
				printFail(fmt.Sprintf("Push failed for %s: %v", t.Slug, err))
				continue
			}
			url, err := gh.CreatePR(gh.PROptions{
				Slug:     t.Slug,
				Branch:   entries[i].Branch,
				Task:     t.Description,
				LogPath:  filepath.Join(cfg.LogDir, t.Slug+".log"),
				RepoRoot: repoRoot,
			})
			if err != nil {
				printFail(fmt.Sprintf("PR failed for %s: %v", t.Slug, err))
			} else {
				printSuccess(fmt.Sprintf("%-30s %s", t.Slug, url))
			}
		}
	}

	// ── 8. Cleanup worktrees ───────────────────────────────────────────────
	if !cfg.KeepWorktrees {
		printSection("Cleaning up worktrees...")
		for _, t := range tasks {
			if err := wm.Destroy(t.Slug); err != nil {
				printWarn(fmt.Sprintf("cleanup failed for %s: %v", t.Slug, err))
			}
		}
	}

	// ── 9. Summary ─────────────────────────────────────────────────────────
	printSummary(results)

	// Exit non-zero if any task failed (CI-compatible)
	for _, r := range results {
		if !r.Success {
			os.Exit(1)
		}
	}

	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func resolveTaskFile(cfg config.Config) (path string, cleanup func(), err error) {
	if cfg.IssueNumber > 0 {
		repoRoot, _ := os.Getwd()
		tmp, fetchErr := gh.FetchIssueTasks(cfg.IssueNumber, repoRoot)
		if fetchErr != nil {
			return "", nil, fmt.Errorf("failed to fetch GitHub issue #%d: %w", cfg.IssueNumber, fetchErr)
		}
		return tmp, func() { os.Remove(tmp) }, nil
	}
	return cfg.PRDFile, nil, nil
}

func filterBySlug(tasks []parser.Task, slug string) []parser.Task {
	for _, t := range tasks {
		if t.Slug == slug {
			return []parser.Task{t}
		}
	}
	return nil
}

func slugList(tasks []parser.Task) string {
	parts := make([]string, len(tasks))
	for i, t := range tasks {
		parts[i] = t.Slug
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func statusStr(success bool) string {
	if success {
		return "done"
	}
	return "failed"
}

func printDryRun(tasks []parser.Task, cfg config.Config) error {
	fmt.Println(yellow("\n[RALPHY DRY RUN] The following would be executed:\n"))
	for i, t := range tasks {
		fmt.Printf("  Task %d: %q\n", i+1, t.Description)
		fmt.Printf("    Branch:   %s/%s\n", cfg.BranchPrefix, t.Slug)
		fmt.Printf("    Worktree: %s/%s\n", cfg.WorktreeDir, t.Slug)
		fmt.Printf("    Model:    %s\n", t.Model)
		fmt.Printf("    Log:      %s/%s.log\n\n", cfg.LogDir, t.Slug)
	}
	fmt.Println(yellow("No changes made."))
	return nil
}

func printSummary(results []agent.Result) {
	succeeded, failed := 0, 0
	for _, r := range results {
		if r.Success {
			succeeded++
		} else {
			failed++
		}
	}
	fmt.Println()
	fmt.Println(bold("─────────────────────────────────────────────────"))
	line := fmt.Sprintf("[RALPHY] Run complete: %d succeeded, %d failed", succeeded, failed)
	if failed == 0 {
		fmt.Println(green(line))
	} else {
		fmt.Println(red(line))
	}
	fmt.Println(bold("─────────────────────────────────────────────────"))
}

func printResult(r agent.Result) {
	if r.Success {
		printSuccess(fmt.Sprintf("%-30s done  (%.0fs)", r.Slug, r.Duration.Seconds()))
	} else {
		printFail(fmt.Sprintf("%-30s FAILED (%.0fs) — see %s", r.Slug, r.Duration.Seconds(), r.LogPath))
	}
}

// ── Terminal colors ────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	cRed   = "\033[31m"
	cGreen = "\033[32m"
	cYellow = "\033[33m"
	cBlue  = "\033[34m"
	cBold  = "\033[1m"
)

func red(s string) string    { return cRed + s + reset }
func green(s string) string  { return cGreen + s + reset }
func yellow(s string) string { return cYellow + s + reset }
func blue(s string) string   { return cBlue + s + reset }
func bold(s string) string   { return cBold + s + reset }

func printBanner() {
	fmt.Println(bold(blue("\n╔══════════════════════════════════════════════╗")))
	fmt.Println(bold(blue("║   RALPHY — Multi-Task AI Coding Orchestrator  ║")))
	fmt.Println(bold(blue("╚══════════════════════════════════════════════╝\n")))
}

func printSection(s string) {
	fmt.Printf("\n%s %s\n", bold("[RALPHY]"), s)
}

func printInfo(s string) {
	fmt.Printf("  %s\n", s)
}

func printSuccess(s string) {
	fmt.Printf("  %s %s\n", green("✓"), s)
}

func printFail(s string) {
	fmt.Printf("  %s %s\n", red("✗"), s)
}

func printWarn(s string) {
	fmt.Printf("  %s %s\n", yellow("⚠"), s)
}
