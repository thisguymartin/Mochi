package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/config"
	gh "github.com/thisguymartin/ai-forge/internal/github"
	"github.com/thisguymartin/ai-forge/internal/memory"
	"github.com/thisguymartin/ai-forge/internal/output"
	"github.com/thisguymartin/ai-forge/internal/parser"
	"github.com/thisguymartin/ai-forge/internal/reviewer"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

// LoopResult captures the outcome of a full Ralph Loop run for one task.
type LoopResult struct {
	FinalWorkerResult agent.Result
	Iterations        int
	FinalMemory       memory.Context
}

// Run is the main entry point for a MOCHI execution cycle.
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

	// ── 6. Invoke agents (via Ralph Loop) ──────────────────────────────────
	printSection("Invoking agents...")
	results := make([]agent.Result, len(tasks))
	loopResults := make([]LoopResult, len(tasks))

	if cfg.Sequential {
		for i, t := range tasks {
			printInfo(fmt.Sprintf("⟳  %-28s [%s]", t.Slug, t.Model))
			_ = wm.UpdateStatus(t.Slug, "running")
			loopResults[i] = runRalphLoop(cfg, t, entries[i])
			results[i] = loopResults[i].FinalWorkerResult
			_ = wm.UpdateStatus(t.Slug, statusStr(results[i].Success))
			printLoopResult(loopResults[i])
		}
	} else {
		var wg sync.WaitGroup
		for i, t := range tasks {
			wg.Add(1)
			go func(idx int, task parser.Task, entry *worktree.Entry) {
				defer wg.Done()
				printInfo(fmt.Sprintf("⟳  %-28s [%s]", task.Slug, task.Model))
				_ = wm.UpdateStatus(task.Slug, "running")
				loopResults[idx] = runRalphLoop(cfg, task, entry)
				results[idx] = loopResults[idx].FinalWorkerResult
				_ = wm.UpdateStatus(task.Slug, statusStr(results[idx].Success))
				printLoopResult(loopResults[idx])
			}(i, t, entries[i])
		}
		wg.Wait()
	}

	// ── 7. Post-loop output dispatch ───────────────────────────────────────
	if cfg.OutputMode != "" && cfg.OutputMode != string(output.ModePR) {
		printSection(fmt.Sprintf("Writing output (%s)...", cfg.OutputMode))
		for i, t := range tasks {
			if !results[i].Success {
				printWarn(fmt.Sprintf("Skipping output for %-24s (agent failed)", t.Slug))
				continue
			}
			if err := output.Handle(output.Options{
				Mode:         output.Mode(cfg.OutputMode),
				Task:         t,
				Entry:        entries[i],
				WorkerResult: results[i],
				MemCtx:       loopResults[i].FinalMemory,
				Iterations:   loopResults[i].Iterations,
				OutputDir:    cfg.OutputDir,
				RepoRoot:     repoRoot,
			}); err != nil {
				printFail(fmt.Sprintf("Output failed for %s: %v", t.Slug, err))
			} else {
				printSuccess(fmt.Sprintf("%-30s written to %s/", t.Slug, cfg.OutputDir))
			}
		}
	}

	// ── 8. Create PRs ──────────────────────────────────────────────────────
	if cfg.CreatePRs && cfg.OutputMode == string(output.ModePR) {
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
			// Use the last iteration log if available, else fallback to base slug
			logPath := filepath.Join(cfg.LogDir, t.Slug+".log")
			if loopResults[i].Iterations > 1 {
				logPath = filepath.Join(cfg.LogDir, fmt.Sprintf("%s-iter%d.log", t.Slug, loopResults[i].Iterations))
			}
			url, err := gh.CreatePR(gh.PROptions{
				Slug:     t.Slug,
				Branch:   entries[i].Branch,
				Task:     t.Description,
				LogPath:  logPath,
				RepoRoot: repoRoot,
			})
			if err != nil {
				printFail(fmt.Sprintf("PR failed for %s: %v", t.Slug, err))
			} else {
				printSuccess(fmt.Sprintf("%-30s %s", t.Slug, url))
			}
		}
	}

	// ── 9. Cleanup worktrees ───────────────────────────────────────────────
	if !cfg.KeepWorktrees {
		printSection("Cleaning up worktrees...")
		for _, t := range tasks {
			if err := wm.Destroy(t.Slug); err != nil {
				printWarn(fmt.Sprintf("cleanup failed for %s: %v", t.Slug, err))
			}
		}
	}

	// ── 10. Summary ────────────────────────────────────────────────────────
	printSummary(results)

	// Exit non-zero if any task failed (CI-compatible)
	for _, r := range results {
		if !r.Success {
			os.Exit(1)
		}
	}

	return nil
}

// loopEnabled returns true when the Ralph Loop should run more than once
// or when a reviewer is configured.
func loopEnabled(cfg config.Config) bool {
	return cfg.ReviewerModel != "" || cfg.MaxIterations > 1
}

// runRalphLoop executes the worker (and optionally reviewer) loop for a single task.
// With default config (MaxIterations=1, no ReviewerModel) it behaves identically to
// the previous single-pass agent.Invoke call.
func runRalphLoop(cfg config.Config, task parser.Task, entry *worktree.Entry) LoopResult {
	maxIter := cfg.MaxIterations
	if maxIter < 1 {
		maxIter = 1
	}

	var lastResult agent.Result
	var lastMemCtx memory.Context
	iterations := 0

	for iter := 1; iter <= maxIter; iter++ {
		iterations = iter

		// Load memory from previous iteration (empty on first pass)
		memCtx := memory.Load(entry.Path)
		lastMemCtx = memCtx

		if cfg.Verbose && loopEnabled(cfg) {
			printInfo(fmt.Sprintf("  [loop] %s iteration %d/%d", task.Slug, iter, maxIter))
		}

		// Run worker agent
		result := agent.Invoke(agent.InvokeOptions{
			WorktreePath:  entry.Path,
			Task:          task.Description,
			Model:         task.Model,
			Timeout:       cfg.Timeout,
			LogDir:        cfg.LogDir,
			Verbose:       cfg.Verbose,
			Iteration:     iter,
			MaxIterations: maxIter,
			MemoryContext: memCtx,
		}, task.Slug)
		lastResult = result

		// Determine status for memory write
		status := "in-progress"
		if !result.Success {
			status = "failed"
		}

		reviewerNotes := ""
		done := false

		// Run reviewer if configured and worker succeeded
		if cfg.ReviewerModel != "" && result.Success {
			decision, err := reviewer.Review(reviewer.Options{
				WorktreePath: entry.Path,
				Task:         task.Description,
				Model:        cfg.ReviewerModel,
				WorkerOutput: result.Output,
				Iteration:    iter,
				MaxIter:      maxIter,
				Timeout:      cfg.Timeout,
				Verbose:      cfg.Verbose,
				LogDir:       cfg.LogDir,
			})
			if err != nil {
				printWarn(fmt.Sprintf("reviewer error for %s iter %d: %v", task.Slug, iter, err))
			} else {
				reviewerNotes = decision.Feedback
				done = decision.Done
			}
		}

		if result.Success && cfg.ReviewerModel == "" {
			done = true
		}
		if !result.Success {
			done = true // stop on agent failure
		}

		if done || iter == maxIter {
			if done && result.Success {
				status = "done"
			}
		}

		// Write memory files after each iteration
		_ = memory.Write(entry.Path, memory.IterationData{
			Iteration:     iter,
			Task:          task.Description,
			WorkerOutput:  result.Output,
			ReviewerNotes: reviewerNotes,
			Status:        status,
		})

		// Reload memory context so LoopResult reflects latest state
		lastMemCtx = memory.Load(entry.Path)

		if done {
			break
		}
	}

	return LoopResult{
		FinalWorkerResult: lastResult,
		Iterations:        iterations,
		FinalMemory:       lastMemCtx,
	}
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
	fmt.Println(yellow("\n[MOCHI DRY RUN] The following would be executed:\n"))
	for i, t := range tasks {
		fmt.Printf("  Task %d: %q\n", i+1, t.Description)
		fmt.Printf("    Branch:      %s/%s\n", cfg.BranchPrefix, t.Slug)
		fmt.Printf("    Worktree:    %s/%s\n", cfg.WorktreeDir, t.Slug)
		fmt.Printf("    Model:       %s\n", t.Model)
		fmt.Printf("    Log:         %s/%s.log\n", cfg.LogDir, t.Slug)
		if cfg.ReviewerModel != "" {
			fmt.Printf("    Reviewer:    %s (max %d iterations)\n", cfg.ReviewerModel, cfg.MaxIterations)
		}
		fmt.Printf("    Output mode: %s\n\n", cfg.OutputMode)
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
	line := fmt.Sprintf("[MOCHI] Run complete: %d succeeded, %d failed", succeeded, failed)
	if failed == 0 {
		fmt.Println(green(line))
	} else {
		fmt.Println(red(line))
	}
	fmt.Println(bold("─────────────────────────────────────────────────"))
}

func printLoopResult(lr LoopResult) {
	r := lr.FinalWorkerResult
	if r.Success {
		if lr.Iterations > 1 {
			printSuccess(fmt.Sprintf("%-30s done  (%.0fs, %d iterations)", r.Slug, r.Duration.Seconds(), lr.Iterations))
		} else {
			printSuccess(fmt.Sprintf("%-30s done  (%.0fs)", r.Slug, r.Duration.Seconds()))
		}
	} else {
		printFail(fmt.Sprintf("%-30s FAILED (%.0fs) — see %s", r.Slug, r.Duration.Seconds(), r.LogPath))
	}
}

// ── Terminal colors ────────────────────────────────────────────────────────

const (
	reset   = "\033[0m"
	cRed    = "\033[31m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cBlue   = "\033[34m"
	cBold   = "\033[1m"
)

func red(s string) string    { return cRed + s + reset }
func green(s string) string  { return cGreen + s + reset }
func yellow(s string) string { return cYellow + s + reset }
func blue(s string) string   { return cBlue + s + reset }
func bold(s string) string   { return cBold + s + reset }

func printBanner() {
	fmt.Println(bold(blue("\n╔══════════════════════════════════════════════╗")))
	fmt.Println(bold(blue("║    MOCHI — Multi-Task AI Coding Orchestrator  ║")))
	fmt.Println(bold(blue("╚══════════════════════════════════════════════╝\n")))
}

func printSection(s string) {
	fmt.Printf("\n%s %s\n", bold("[MOCHI]"), s)
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
