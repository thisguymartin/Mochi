package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/config"
	gh "github.com/thisguymartin/ai-forge/internal/github"
	"github.com/thisguymartin/ai-forge/internal/memory"
	"github.com/thisguymartin/ai-forge/internal/output"
	"github.com/thisguymartin/ai-forge/internal/provider"
	"github.com/thisguymartin/ai-forge/internal/reviewer"
	"github.com/thisguymartin/ai-forge/internal/source"
	"github.com/thisguymartin/ai-forge/internal/tui"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

// ErrTasksFailed is returned when one or more tasks did not complete successfully.
// The run summary is already printed to the terminal; callers should not echo this error.
var ErrTasksFailed = errors.New("one or more tasks failed")

// LoopResult captures the outcome of a full Ralph Loop run for one task.
type LoopResult struct {
	FinalWorkerResult agent.Result
	Iterations        int
	FinalMemory       memory.Context
}

// lookPath is a package-level variable wrapping exec.LookPath so tests can swap it.
var lookPath = exec.LookPath

// Orchestrator runs the MOCHI execution cycle.
// Construct with New, optionally call WithSource, then call Run.
type Orchestrator struct {
	cfg    config.Config
	source source.Source // nil = auto-resolve from cfg.InputFile
}

// New creates an Orchestrator with the given configuration.
func New(cfg config.Config) *Orchestrator {
	return &Orchestrator{cfg: cfg}
}

// WithSource overrides the task source, bypassing --input flag resolution.
// Implement source.Source to inject custom sources (MCP, API, database, etc.).
func (o *Orchestrator) WithSource(s source.Source) *Orchestrator {
	o.source = s
	return o
}

// Run is a convenience wrapper for New(cfg).Run().
func Run(cfg config.Config) error {
	return New(cfg).Run()
}

// checkDependencies verifies that all required external tools are present in PATH.
// It always checks for git; checks the model provider CLI based on model prefix;
// and checks gh when --create-prs is used.
// Returns a combined error listing all missing tools with install hints.
func checkDependencies(cfg config.Config) error {
	type tool struct {
		name    string
		install string
	}

	var needed []tool

	needed = append(needed, tool{"git", "https://git-scm.com"})

	providerTool := provider.ToolForModel(cfg.Model)
	needed = append(needed, tool{providerTool.Name, providerTool.Install})

	if cfg.CreatePRs {
		needed = append(needed, tool{"gh", "https://cli.github.com"})
	}

	var missing []tool
	for _, t := range needed {
		if _, err := lookPath(t.name); err != nil {
			missing = append(missing, t)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	names := make([]string, len(missing))
	for i, t := range missing {
		names[i] = t.name
	}
	msg := fmt.Sprintf("missing required tools: %s", strings.Join(names, ", "))
	for _, t := range missing {
		msg += fmt.Sprintf("\n  → install %s from %s", t.name, t.install)
	}
	return fmt.Errorf("%s", msg)
}

// Run executes the full MOCHI cycle.
func (o *Orchestrator) Run() error {
	cfg := o.cfg
	tui.SetVerbose(cfg.Verbose)

	// ── 1. Dependency checks ───────────────────────────────────────────────
	if err := checkDependencies(cfg); err != nil {
		return err
	}

	// ── 2. Resolve source and fetch tasks ─────────────────────────────────
	src := o.source
	if src == nil {
		var err error
		src, err = ResolveSource(cfg)
		if err != nil {
			return err
		}
	}

	tasks, err := src.FetchTasks()
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

	// ── 3. Generate better slugs via AI ────────────────────────────────────
	var needsAiSlug bool
	for _, t := range tasks {
		if len(t.Slug) >= 50 {
			needsAiSlug = true
			break
		}
	}

	if needsAiSlug {
		printSection("Refining branch titles...")
		var slugWg sync.WaitGroup
		slugCtx := context.Background()

		for i := range tasks {
			if len(tasks[i].Slug) >= 50 {
				slugWg.Add(1)
				go func(idx int) {
					defer slugWg.Done()

					promptContext := tasks[idx].Title
					if tasks[idx].Description != "" {
						promptContext += "\n\n" + tasks[idx].Description
					}

					newSlug, err := agent.GenerateTitle(slugCtx, tasks[idx].Model, promptContext)
					if err == nil && newSlug != "" {
						tasks[idx].Slug = newSlug
					} else if cfg.Verbose {
						printWarn(fmt.Sprintf("Failed to generate AI title for task %d: %v", idx+1, err))
					}
				}(i)
			}
		}
		slugWg.Wait()
	}

	printSection(fmt.Sprintf("Found %d task(s): %s", len(tasks), slugList(tasks)))

	// ── 4. Dry run ─────────────────────────────────────────────────────────
	if cfg.DryRun {
		return printDryRun(tasks, cfg)
	}

	// ── 5. Setup ───────────────────────────────────────────────────────────
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("cannot create log dir %q: %w", cfg.LogDir, err)
	}

	wm := worktree.NewManager(repoRoot, cfg.BaseBranch, cfg.BranchPrefix, cfg.WorktreeDir)

	// ── 6. Create worktrees ────────────────────────────────────────────────
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

	// ── 7. Invoke agents (via Ralph Loop) ──────────────────────────────────
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
		// Semaphore channel limits concurrent worktrees when --worktrees N is set.
		var sem chan struct{}
		if cfg.MaxWorktrees > 0 && cfg.MaxWorktrees < len(tasks) {
			sem = make(chan struct{}, cfg.MaxWorktrees)
			printInfo(fmt.Sprintf("Concurrency limited to %d worktree(s)", cfg.MaxWorktrees))
		}

		var wg sync.WaitGroup
		for i, t := range tasks {
			wg.Add(1)
			go func(idx int, task source.Task, entry *worktree.Entry) {
				defer wg.Done()
				if sem != nil {
					sem <- struct{}{}        // acquire
					defer func() { <-sem }() // release
				}
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

	// ── 8. Post-loop output dispatch ────────────────────────────────────────
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

	// ── 9. Create PRs ───────────────────────────────────────────────────────
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
			logPath := filepath.Join(cfg.LogDir, t.Slug+".log")
			if loopResults[i].Iterations > 1 {
				logPath = filepath.Join(cfg.LogDir, fmt.Sprintf("%s-iter%d.log", t.Slug, loopResults[i].Iterations))
			}
			url, err := gh.CreatePR(gh.PROptions{
				Slug:     t.Slug,
				Branch:   entries[i].Branch,
				Task:     t.Title,
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

	// ── 10. Cleanup worktrees ──────────────────────────────────────────────
	if !cfg.KeepWorktrees {
		printSection("Cleaning up worktrees...")
		for _, t := range tasks {
			if err := wm.Destroy(t.Slug); err != nil {
				printWarn(fmt.Sprintf("cleanup failed for %s: %v", t.Slug, err))
			}
		}
	}

	// ── 11. Summary ────────────────────────────────────────────────────────
	printSummary(results)

	// ── 12. Launch Grove if requested ──────────────────────────────────────
	if cfg.LaunchGrove {
		grovePath, err := exec.LookPath("grove")
		if err != nil {
			printWarn("grove not found in PATH — skipping Grove launch")
			printInfo("Install Grove from https://github.com/thisguymartin/grove")
		} else {
			printSection("Launching Grove workspace...")
			cmd := exec.Command(grovePath, repoRoot)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				printWarn(fmt.Sprintf("Grove exited with error: %v", err))
			}
		}
	}

	// Return ErrTasksFailed for CI — the summary is already printed above.
	for _, r := range results {
		if !r.Success {
			return ErrTasksFailed
		}
	}
	return nil
}

// ResolveSource determines which Source to use based on the config flags.
func ResolveSource(cfg config.Config) (source.Source, error) {
	inputFile := cfg.InputFile
	if inputFile == "" {
		return nil, fmt.Errorf("no task source specified — use --input <path>")
	}

	// Auto-detect common task file names if the default is missing
	if inputFile == "PRD.md" {
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			candidates := []string{
				"PLAN.md", "plan.md", "input.md", "tasks.md",
				"docs/PLAN.md", "docs/PRD.md", "examples/PRD.md",
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					inputFile = c
					break
				}
			}
		}
	}

	return &source.FileSource{Path: inputFile}, nil
}

// runRalphLoop executes the worker (and optionally reviewer) loop for a single task.
func runRalphLoop(cfg config.Config, task source.Task, entry *worktree.Entry) LoopResult {
	maxIter := cfg.MaxIterations
	if maxIter < 1 {
		maxIter = 1
	}

	var lastResult agent.Result
	var lastMemCtx memory.Context
	iterations := 0

	for iter := 1; iter <= maxIter; iter++ {
		iterations = iter

		memCtx := memory.Load(entry.Path)
		lastMemCtx = memCtx

		tui.Logger.Debug("loop iteration", "task", task.Slug, "iter", iter, "max", maxIter)

		fullTaskContext := task.Title
		if task.Description != "" {
			fullTaskContext += "\n\n" + task.Description
		}

		result := agent.Invoke(agent.InvokeOptions{
			WorktreePath:  entry.Path,
			Task:          fullTaskContext,
			Model:         task.Model,
			Timeout:       cfg.Timeout,
			LogDir:        cfg.LogDir,
			Verbose:       cfg.Verbose,
			Iteration:     iter,
			MaxIterations: maxIter,
			MemoryContext: memCtx,
		}, task.Slug)
		lastResult = result

		status := "in-progress"
		if !result.Success {
			status = "failed"
		}

		reviewerNotes := ""
		done := false

		if cfg.ReviewerModel != "" && result.Success {
			decision, err := reviewer.Review(reviewer.Options{
				WorktreePath: entry.Path,
				Task:         fullTaskContext,
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
			done = true
		}

		if done || iter == maxIter {
			if done && result.Success {
				status = "done"
			}
		}

		_ = memory.Write(entry.Path, memory.IterationData{
			Iteration:     iter,
			Task:          fullTaskContext,
			WorkerOutput:  result.Output,
			ReviewerNotes: reviewerNotes,
			Status:        status,
		})

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
