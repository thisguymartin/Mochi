package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/orchestrator"
	"github.com/thisguymartin/ai-forge/internal/tui"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

const Version = "0.1.0"

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "mochi",
	Short: "Multi-task AI coding orchestrator",
	Long: `MOCHI reads a task file (PRD.md or a GitHub Issue), spins up isolated
git worktrees for each task, invokes an AI agent in each worktree in parallel,
and optionally opens GitHub pull requests for every completed task.

Supported providers (auto-detected from model name):
  claude  claude-opus-4-6 | claude-sonnet-4-6 | claude-haiku-4-5
  gemini  gemini-2.5-pro  | gemini-2.0-flash   | gemini-1.5-pro`,
	Example: `  # Run all tasks with the default Claude model
  mochi --prd examples/PRD.md

  # Run with Gemini 2.5 Pro
  mochi --prd examples/PRD.md --model gemini-2.5-pro

  # Run with Claude Opus and auto-create PRs
  mochi --prd examples/PRD.md --model claude-opus-4-6 --create-prs

  # Pull tasks from GitHub Issue #88 and create PRs
  mochi --issue 88 --create-prs

  # Preview what would happen without making any changes
  mochi --prd examples/PRD.md --dry-run

  # Debug a single task sequentially with live output
  mochi --prd examples/PRD.md --task fix-mobile-navbar --sequential --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no task source was explicitly provided, show the info panel and exit.
		hasInput := cmd.Flags().Changed("prd") || cmd.Flags().Changed("input") || cmd.Flags().Changed("plan")
		if !hasInput && cfg.IssueNumber == 0 {
			cwd, _ := os.Getwd()
			tui.PrintInfo(Version, cfg.Model, cwd)
			return nil
		}

		// Interactive model picker
		if cfg.PromptModel {
			selected, err := tui.RunModelPicker(cfg.Model)
			if err != nil {
				return fmt.Errorf("model picker: %w", err)
			}
			cfg.Model = selected
		}

		tui.RunSplash()
		return orchestrator.Run(cfg)
	},
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale worktree registrations and manifest entries",
	Long: `Runs 'git worktree prune' to clear git's stale registrations, then removes
any manifest entries whose paths no longer exist on disk.

Use this after a crashed or interrupted run leaves orphaned worktree state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)
		pruned, err := wm.Prune()
		if err != nil {
			return err
		}
		if len(pruned) == 0 {
			fmt.Println("Nothing to prune.")
		} else {
			for _, slug := range pruned {
				fmt.Printf("  pruned  %s\n", slug)
			}
			fmt.Printf("Pruned %d stale worktree(s).\n", len(pruned))
		}
		return nil
	},
}

// Execute is the entry point called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	defaults := config.Default()

	// Input source
	rootCmd.Flags().StringVarP(&cfg.InputFile, "input", "i", defaults.InputFile,
		"Path to the task file (markdown with a '## Tasks' section)")
	rootCmd.Flags().StringVar(&cfg.InputFile, "prd", defaults.InputFile,
		"Alias for --input")
	rootCmd.Flags().StringVar(&cfg.InputFile, "plan", defaults.InputFile,
		"Alias for --input")
	rootCmd.Flags().IntVar(&cfg.IssueNumber, "issue", 0,
		"Pull tasks from a GitHub Issue number (requires gh CLI)")

	// Model
	rootCmd.Flags().StringVar(&cfg.Model, "model", defaults.Model,
		"Default model — Claude (claude-opus-4-6 | claude-sonnet-4-6 | claude-haiku-4-5) or Gemini (gemini-2.5-pro | gemini-2.0-flash)")
	rootCmd.Flags().BoolVar(&cfg.PromptModel, "prompt-model", false,
		"Show interactive model picker before running")

	// Execution control
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false,
		"Preview what would run without making any changes")
	rootCmd.Flags().BoolVar(&cfg.Sequential, "sequential", false,
		"Run tasks one at a time instead of in parallel (useful for debugging)")
	rootCmd.Flags().IntVar(&cfg.MaxWorktrees, "worktrees", defaults.MaxWorktrees,
		"Max concurrent worktrees (0 = unlimited, matches task count)")
	rootCmd.Flags().StringVar(&cfg.TaskFilter, "task", "",
		"Run only the task matching this slug (e.g. fix-mobile-navbar)")
	rootCmd.Flags().IntVar(&cfg.Timeout, "timeout", defaults.Timeout,
		"Maximum time in seconds to wait for a single agent")
	rootCmd.Flags().BoolVar(&cfg.Verbose, "verbose", false,
		"Stream agent output live to the terminal in addition to the log file")

	// GitHub
	rootCmd.Flags().BoolVar(&cfg.CreatePRs, "create-prs", false,
		"Push branches and open a GitHub PR for each completed task")

	// Worktree
	rootCmd.Flags().BoolVar(&cfg.KeepWorktrees, "keep-worktrees", false,
		"Keep worktrees on disk after the run (default: remove them)")
	rootCmd.Flags().StringVar(&cfg.BaseBranch, "base-branch", defaults.BaseBranch,
		"Branch to base each worktree on")

	// Workspace (ai-native-dev integration)
	rootCmd.Flags().StringVar(&cfg.Workspace, "workspace", "",
		"Launch ai-native-dev workspace with worktree panes (zellij | auto)")

	// Git  Loop
	rootCmd.Flags().StringVar(&cfg.ReviewerModel, "reviewer-model", "",
		"Model for the reviewer agent — enables the Ralph Loop when set (e.g. claude-opus-4-6)")
	rootCmd.Flags().IntVar(&cfg.MaxIterations, "max-iterations", defaults.MaxIterations,
		"Maximum worker iterations per task (default: 1, no loop)")
	rootCmd.Flags().StringVar(&cfg.OutputMode, "output-mode", defaults.OutputMode,
		"Output mode: pr | research-report | audit | knowledge-base | issue | file")
	rootCmd.Flags().StringVar(&cfg.OutputDir, "output-dir", defaults.OutputDir,
		"Directory for file/report outputs (used with --output-mode file or research-report)")

	// Apply non-flag defaults that don't need user exposure
	cfg.BranchPrefix = defaults.BranchPrefix
	cfg.WorktreeDir = defaults.WorktreeDir
	cfg.LogDir = defaults.LogDir

	rootCmd.AddCommand(pruneCmd)
}
