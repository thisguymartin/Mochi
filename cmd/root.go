package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/orchestrator"
	"github.com/thisguymartin/ai-forge/internal/tui"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

const Version = "0.2.0"

var (
	cfg       config.Config
	forceFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "mochi",
	Short: "Multi-task AI coding orchestrator",
	Long: `MOCHI reads a task source (a file or any custom source plugin), spins up isolated
git worktrees for each task, invokes an AI agent in each worktree in parallel,
and optionally opens GitHub pull requests for every completed task.

Supported providers (auto-detected from model name):
  claude  claude-opus-4-6 | claude-sonnet-4-6 | claude-haiku-4-5
  gemini  gemini-2.5-pro  | gemini-2.0-flash   | gemini-1.5-pro
  codex   gpt-* | o1* | o3* | o4* | codex-*`,
	Example: `  # Run with a file (entire content becomes the task context)
  mochi --input examples/PRD.md

  # Run with Gemini 2.5 Pro (default)
  mochi --input examples/PRD.md

  # Run with Claude Opus and auto-create PRs
  mochi --input examples/PRD.md --model claude-opus-4-6 --create-prs

  # Preview what would happen without making any changes
  mochi --input examples/PRD.md --dry-run

  # Dry-run with concurrency limit
  mochi --input examples/PRD.md --worktrees 2 --dry-run

  # Debug a single task sequentially with live output
  mochi --input examples/PRD.md --task fix-mobile-navbar --sequential --verbose

  # Reviewer loop dry-run
  mochi --input examples/PRD.md --reviewer-model claude-opus-4-6 --max-iterations 3 --dry-run

  # Research report output
  mochi --input examples/PRD.md --output-mode research-report --output-dir ./reports

  # List all tracked worktrees
  mochi worktree list

  # Remove a specific worktree
  mochi worktree remove fix-mobile-navbar

  # Remove all worktrees
  mochi worktree clean

  # Run a command inside a task worktree in a floating pane (tmux)
  mochi pane run --task fix-mobile-navbar --cmd "go test ./..." --mode popup`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no task source was explicitly provided, show the info panel and exit.
		hasInput := cmd.Flags().Changed("prd") || cmd.Flags().Changed("input") || cmd.Flags().Changed("plan")
		if !hasInput {
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

		// --grove implies --keep-worktrees (worktrees must persist for Grove)
		if cfg.LaunchGrove {
			cfg.KeepWorktrees = true
		}

		tui.RunSplash()
		if err := orchestrator.Run(cfg); err != nil {
			if errors.Is(err, orchestrator.ErrTasksFailed) {
				os.Exit(1) // summary already printed; avoid duplicate error message
			}
			return err
		}
		return nil
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
				fmt.Printf("  %s  %s\n", tui.SuccessStyle.Render("pruned"), slug)
			}
			fmt.Printf("Pruned %d stale worktree(s).\n", len(pruned))
		}
		return nil
	},
}

// ── Worktree management subcommands ────────────────────────────────────────

var worktreeCmd = &cobra.Command{
	Use:     "worktree",
	Aliases: []string{"wt"},
	Short:   "Manage MOCHI-tracked worktrees",
	Long:    `Commands for listing, inspecting, and removing MOCHI-tracked git worktrees.`,
}

var wtListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tracked worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		entries, err := wm.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No tracked worktrees.")
			return nil
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{e.Slug, e.Branch, e.Status, e.Path})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(tui.ColorMuted)).
			Headers("SLUG", "BRANCH", "STATUS", "PATH").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return tui.TableHeaderStyle
				}
				s := lipgloss.NewStyle().Padding(0, 1)
				if col == 2 && row >= 0 && row < len(rows) {
					switch rows[row][2] {
					case "done":
						return s.Foreground(tui.ColorSuccess)
					case "failed":
						return s.Foreground(tui.ColorError)
					case "running":
						return s.Foreground(tui.ColorWarning)
					default:
						return s.Foreground(tui.ColorMuted)
					}
				}
				return s
			})

		fmt.Println(t)
		fmt.Printf("\n%d worktree(s) tracked.\n", len(entries))
		return nil
	},
}

var wtStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show worktrees with disk-existence check",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		entries, err := wm.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No tracked worktrees.")
			return nil
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			onDisk := "yes"
			if _, statErr := os.Stat(e.Path); os.IsNotExist(statErr) {
				onDisk = "NO"
			}
			rows = append(rows, []string{e.Slug, e.Status, onDisk, e.Path})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(tui.ColorMuted)).
			Headers("SLUG", "STATUS", "ON DISK", "PATH").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return tui.TableHeaderStyle
				}
				s := lipgloss.NewStyle().Padding(0, 1)
				if row >= 0 && row < len(rows) {
					switch {
					case col == 1: // STATUS
						switch rows[row][1] {
						case "done":
							return s.Foreground(tui.ColorSuccess)
						case "failed":
							return s.Foreground(tui.ColorError)
						case "running":
							return s.Foreground(tui.ColorWarning)
						default:
							return s.Foreground(tui.ColorMuted)
						}
					case col == 2: // ON DISK
						if rows[row][2] == "NO" {
							return s.Foreground(tui.ColorError)
						}
						return s.Foreground(tui.ColorSuccess)
					}
				}
				return s
			})

		fmt.Println(t)
		return nil
	},
}

var wtRemoveCmd = &cobra.Command{
	Use:     "remove [slug...]",
	Aliases: []string{"rm"},
	Short:   "Remove specific worktrees by slug",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		if len(args) > 1 && !forceFlag {
			if !tui.ConfirmDestructive(fmt.Sprintf("Remove %d worktrees?", len(args))) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		var errors []string
		for _, slug := range args {
			if err := wm.Destroy(slug); err != nil {
				errors = append(errors, fmt.Sprintf("  %s: %v", slug, err))
			} else {
				fmt.Printf("  %s  %s\n", tui.SuccessStyle.Render("removed"), slug)
			}
		}
		if len(errors) > 0 {
			for _, e := range errors {
				fmt.Fprintln(os.Stderr, e)
			}
			return fmt.Errorf("failed to remove %d worktree(s)", len(errors))
		}
		return nil
	},
}

var wtCleanCmd = &cobra.Command{
	Use:     "clean",
	Aliases: []string{"nuke"},
	Short:   "Remove ALL tracked worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		if !forceFlag {
			entries, err := wm.List()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("No tracked worktrees to remove.")
				return nil
			}
			if !tui.ConfirmDestructive(fmt.Sprintf("Remove ALL %d tracked worktrees?", len(entries))) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		removed, err := wm.DestroyAll()
		if len(removed) > 0 {
			for _, slug := range removed {
				fmt.Printf("  %s  %s\n", tui.SuccessStyle.Render("removed"), slug)
			}
		}
		if err != nil {
			return err
		}
		if len(removed) == 0 {
			fmt.Println("No tracked worktrees to remove.")
		} else {
			fmt.Printf("Removed %d worktree(s).\n", len(removed))
		}
		return nil
	},
}

var wtAddCmd = &cobra.Command{
	Use:     "add <slug>",
	Aliases: []string{"create", "new"},
	Short:   "Create a new worktree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		entry, err := wm.Create(slug)
		if err != nil {
			return err
		}

		fmt.Printf("  %s  %s\n", tui.SuccessStyle.Render("created"), slug)
		fmt.Printf("  Path: %s\n", entry.Path)
		fmt.Printf("  Branch: %s\n", entry.Branch)
		return nil
	},
}

var wtEnterCmd = &cobra.Command{
	Use:     "enter <slug>",
	Aliases: []string{"shell", "cd"},
	Short:   "Open a shell in the worktree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		entry, err := wm.GetEntry(slug)
		if err != nil {
			return err
		}

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}

		c := exec.Command(shell)
		c.Dir = entry.Path
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		fmt.Printf("Entering %s...\n", entry.Path)
		return c.Run()
	},
}

var wtZellijCmd = &cobra.Command{
	Use:   "zellij",
	Short: "Launch Zellij with tabs for all worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		zellijPath, err := exec.LookPath("zellij")
		if err != nil {
			return fmt.Errorf("zellij not found in PATH")
		}

		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)

		entries, err := wm.List()
		if err != nil {
			return err
		}

		layoutContent := generateZellijLayout(repoRoot, entries)
		tmpFile, err := os.CreateTemp("", "mochi-layout-*.kdl")
		if err != nil {
			return fmt.Errorf("failed to create temp layout file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(layoutContent); err != nil {
			return fmt.Errorf("failed to write layout file: %w", err)
		}
		tmpFile.Close()

		c := exec.Command(zellijPath, "--layout", tmpFile.Name())
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		return c.Run()
	},
}

func generateZellijLayout(repoRoot string, entries []worktree.Entry) string {
	layout := `layout {
    default_tab_template {
        pane size=1 borderless=true {
            plugin location="zellij:tab-bar"
        }
        children
        pane size=2 borderless=true {
            plugin location="zellij:status-bar"
        }
    }
`
	// Add ROOT tab
	layout += fmt.Sprintf("    tab name=\"ROOT\" cwd=\"%s\" {\n        pane\n    }\n", repoRoot)

	// Add worktree tabs
	for _, entry := range entries {
		layout += fmt.Sprintf("    tab name=\"%s\" cwd=\"%s\" {\n        pane\n    }\n", entry.Slug, entry.Path)
	}
	layout += "}\n"
	return layout
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
		"Path to the task file (entire content becomes task context)")
	rootCmd.Flags().StringVar(&cfg.InputFile, "prd", defaults.InputFile,
		"Alias for --input")
	rootCmd.Flags().StringVar(&cfg.InputFile, "plan", defaults.InputFile,
		"Alias for --input")

	// Model
	rootCmd.Flags().StringVar(&cfg.Model, "model", defaults.Model,
		"Default model — Claude (claude-*), Gemini (gemini-*), or Codex (gpt-*, o1/o3/o4, codex-*)")
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
	rootCmd.Flags().BoolVar(&cfg.LaunchGrove, "grove", false,
		"Launch Grove workspace after run completes (implies --keep-worktrees)")
	rootCmd.Flags().StringVar(&cfg.BaseBranch, "base-branch", defaults.BaseBranch,
		"Branch to base each worktree on")

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

	// Custom glamour-rendered help for root command only
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}
		helpMD := buildHelpMarkdown(cmd)
		if tui.IsTTY() {
			r, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(80),
			)
			if err == nil {
				rendered, err := r.Render(helpMD)
				if err == nil {
					fmt.Print(rendered)
					return
				}
			}
		}
		// Plain fallback for CI/pipes or render errors
		defaultHelp(cmd, args)
	})

	// Force flag on destructive commands
	wtRemoveCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompt")
	wtCleanCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompt")

	// Subcommands
	rootCmd.AddCommand(pruneCmd)

	worktreeCmd.AddCommand(wtListCmd)
	worktreeCmd.AddCommand(wtStatusCmd)
	worktreeCmd.AddCommand(wtRemoveCmd)
	worktreeCmd.AddCommand(wtCleanCmd)
	worktreeCmd.AddCommand(wtAddCmd)
	worktreeCmd.AddCommand(wtEnterCmd)
	worktreeCmd.AddCommand(wtZellijCmd)
	rootCmd.AddCommand(worktreeCmd)

	rootCmd.AddCommand(paneCmd)
}
