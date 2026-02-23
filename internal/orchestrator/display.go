package orchestrator

import (
	"fmt"
	"strings"

	"github.com/thisguymartin/ai-forge/internal/agent"
	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/source"
	"github.com/thisguymartin/ai-forge/internal/tui"
)

func printSection(s string) {
	fmt.Printf("\n%s %s\n", tui.BoldStyle.Render("[MOCHI]"), s)
}

func printInfo(s string) {
	fmt.Printf("  %s\n", s)
}

func printSuccess(s string) {
	fmt.Printf("  %s %s\n", tui.SuccessStyle.Render("✓"), s)
}

func printFail(s string) {
	fmt.Printf("  %s %s\n", tui.ErrorStyle.Render("✗"), s)
}

func printWarn(s string) {
	fmt.Printf("  %s %s\n", tui.WarningStyle.Render("⚠"), s)
}

func printDryRun(tasks []source.Task, cfg config.Config) error {
	var b strings.Builder

	b.WriteString(tui.WarningStyle.Render("DRY RUN") + " — the following would be executed:\n\n")

	// Config summary
	keyStyle := tui.MutedStyle
	if cfg.Sequential {
		b.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("Execution mode:          "), "sequential"))
	} else {
		b.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("Execution mode:          "), "parallel"))
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("Base branch:             "), cfg.BaseBranch))
	b.WriteString(fmt.Sprintf("  %s  %v\n", keyStyle.Render("Create PRs:              "), cfg.CreatePRs))
	b.WriteString(fmt.Sprintf("  %s  %v\n", keyStyle.Render("Keep worktrees:          "), cfg.KeepWorktrees))
	if cfg.MaxWorktrees > 0 {
		b.WriteString(fmt.Sprintf("  %s  %d\n", keyStyle.Render("Max concurrent worktrees:"), cfg.MaxWorktrees))
	}
	b.WriteString(fmt.Sprintf("  %s  %d\n", keyStyle.Render("Total tasks:             "), len(tasks)))
	b.WriteString("\n")

	// Tasks
	for i, t := range tasks {
		header := tui.BoldStyle.Render(fmt.Sprintf("Task %d: %q", i+1, t.Title))
		b.WriteString("  " + header + "\n")
		b.WriteString(fmt.Sprintf("    %s  %s/%s\n", keyStyle.Render("Branch:   "), cfg.BranchPrefix, t.Slug))
		b.WriteString(fmt.Sprintf("    %s  %s/%s\n", keyStyle.Render("Worktree: "), cfg.WorktreeDir, t.Slug))
		b.WriteString(fmt.Sprintf("    %s  %s\n", keyStyle.Render("Model:    "), t.Model))
		b.WriteString(fmt.Sprintf("    %s  %s/%s.log\n", keyStyle.Render("Log:      "), cfg.LogDir, t.Slug))
		if cfg.ReviewerModel != "" {
			b.WriteString(fmt.Sprintf("    %s  %s (max %d iterations)\n", keyStyle.Render("Reviewer: "), cfg.ReviewerModel, cfg.MaxIterations))
			for iter := 1; iter <= cfg.MaxIterations; iter++ {
				b.WriteString(fmt.Sprintf("    %s  %s/%s-iter%d.log\n", keyStyle.Render(fmt.Sprintf("Log (i%d):", iter)), cfg.LogDir, t.Slug, iter))
			}
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n", keyStyle.Render("Output:   "), cfg.OutputMode))
		b.WriteString("\n")
	}

	b.WriteString(tui.WarningStyle.Render("No changes made."))

	fmt.Println()
	fmt.Println(tui.DryRunBoxStyle.Render(b.String()))
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

	line := fmt.Sprintf("[MOCHI] Run complete: %d succeeded, %d failed", succeeded, failed)

	style := tui.SummaryBoxStyle
	if failed == 0 {
		style = style.BorderForeground(tui.ColorSuccess)
		line = tui.SuccessStyle.Render(line)
	} else {
		style = style.BorderForeground(tui.ColorError)
		line = tui.ErrorStyle.Render(line)
	}

	fmt.Println()
	fmt.Println(style.Render(line))
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
