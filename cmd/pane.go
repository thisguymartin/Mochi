package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/worktree"
)

var (
	paneTask    string
	paneCommand string
	paneMode    string
	paneBackend string
)

var paneCmd = &cobra.Command{
	Use:   "pane",
	Short: "Open task-scoped terminal panes",
	Long:  "Run commands in a task worktree using tmux panes/popups when available.",
}

var paneRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command in a tracked task worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := strings.ToLower(strings.TrimSpace(paneMode))
		backend := strings.ToLower(strings.TrimSpace(paneBackend))

		if mode != "popup" && mode != "split" {
			return fmt.Errorf("invalid --mode %q (allowed: popup|split)", paneMode)
		}
		if backend != "auto" && backend != "tmux" && backend != "shell" {
			return fmt.Errorf("invalid --backend %q (allowed: auto|tmux|shell)", paneBackend)
		}

		repoRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		defaults := config.Default()
		wm := worktree.NewManager(repoRoot, defaults.BaseBranch, defaults.BranchPrefix, defaults.WorktreeDir)
		entry, err := wm.GetEntry(paneTask)
		if err != nil {
			return fmt.Errorf("%w\nhint: run `mochi worktree list` to find valid task slugs", err)
		}

		switch backend {
		case "shell":
			return runShellInWorktree(entry.Path, paneCommand)
		case "tmux":
			return runTmux(mode, entry.Path, paneCommand)
		default: // auto
			if hasCmd("tmux") && os.Getenv("TMUX") != "" {
				if err := runTmux(mode, entry.Path, paneCommand); err == nil {
					return nil
				}
			}
			fmt.Println("tmux pane unavailable; running command directly in current shell")
			return runShellInWorktree(entry.Path, paneCommand)
		}
	},
}

func init() {
	paneRunCmd.Flags().StringVar(&paneTask, "task", "", "Task slug from the MOCHI manifest")
	paneRunCmd.Flags().StringVar(&paneCommand, "cmd", "", "Command to run inside the task worktree")
	paneRunCmd.Flags().StringVar(&paneMode, "mode", "popup", "Pane mode: popup | split")
	paneRunCmd.Flags().StringVar(&paneBackend, "backend", "auto", "Backend: auto | tmux | shell")

	_ = paneRunCmd.MarkFlagRequired("task")
	_ = paneRunCmd.MarkFlagRequired("cmd")

	paneCmd.AddCommand(paneRunCmd)
}

// shellBin returns the user's preferred shell, falling back to /bin/sh.
func shellBin() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

func runTmux(mode, worktreePath, command string) error {
	if !hasCmd("tmux") {
		return fmt.Errorf("tmux is not installed")
	}
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("tmux backend requires running inside an active tmux session")
	}

	shell := shellBin()
	var tmuxCmd *exec.Cmd
	if mode == "split" {
		tmuxCmd = exec.Command("tmux", "split-window", "-c", worktreePath, shell, "-lc", command)
	} else {
		tmuxCmd = exec.Command("tmux", "display-popup", "-E", "-d", worktreePath, shell, "-lc", command)
	}

	out, err := tmuxCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux command failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runShellInWorktree(worktreePath, command string) error {
	c := exec.Command(shellBin(), "-lc", command)
	c.Dir = worktreePath
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func hasCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
