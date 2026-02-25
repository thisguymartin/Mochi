package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thisguymartin/ai-forge/internal/worktree"
)

// Options configures the workspace launch.
type Options struct {
	Mode    string             // "zellij" or "auto"
	Entries []*worktree.Entry
	Verbose bool
}

// Launch starts the ai-native-dev workspace with one pane per worktree.
// When mode is "zellij", it generates a dynamic KDL layout and launches Zellij.
// When mode is "auto", it detects the available workspace tool.
func Launch(opts Options) error {
	mode := opts.Mode
	if mode == "auto" {
		mode = detectMode()
	}

	switch mode {
	case "zellij":
		return launchZellij(opts)
	default:
		return fmt.Errorf("unsupported workspace mode %q (supported: zellij, auto)", opts.Mode)
	}
}

func detectMode() string {
	if _, err := exec.LookPath("zellij"); err == nil {
		return "zellij"
	}
	return ""
}

// launchZellij generates a dynamic Zellij KDL layout with one pane per worktree
// and launches it in a new terminal process.
func launchZellij(opts Options) error {
	if _, err := exec.LookPath("zellij"); err != nil {
		return fmt.Errorf("zellij not found in PATH — install from https://zellij.dev")
	}

	layout := generateZellijLayout(opts.Entries)

	// Write layout to a temp file
	tmpDir := os.TempDir()
	layoutPath := filepath.Join(tmpDir, "mochi-workspace.kdl")
	if err := os.WriteFile(layoutPath, []byte(layout), 0644); err != nil {
		return fmt.Errorf("cannot write Zellij layout: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("  Zellij layout written to %s\n", layoutPath)
	}

	// Launch Zellij in a detached process so it doesn't block Mochi
	cmd := exec.Command("zellij", "--layout", layoutPath, "--session", "mochi-workspace")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch Zellij: %w", err)
	}

	// Detach — don't wait for Zellij to exit
	go func() { _ = cmd.Wait() }()

	fmt.Printf("  Zellij workspace launched (session: mochi-workspace)\n")
	fmt.Printf("  Attach with: zellij attach mochi-workspace\n")
	return nil
}

// generateZellijLayout creates a KDL layout string with:
// - Left column: one pane per worktree showing lazygit
// - Right column: one pane per worktree as a shell in the worktree directory
func generateZellijLayout(entries []*worktree.Entry) string {
	var b strings.Builder

	b.WriteString("layout {\n")
	b.WriteString("  tab name=\"Mochi Worktrees\" {\n")
	b.WriteString("    pane split_direction=\"horizontal\" {\n")

	// Left column: git views
	b.WriteString("      pane split_direction=\"vertical\" size=\"40%\" {\n")
	for _, e := range entries {
		absPath, _ := filepath.Abs(e.Path)
		b.WriteString(fmt.Sprintf("        pane name=\"git: %s\" {\n", e.Slug))
		b.WriteString(fmt.Sprintf("          command \"lazygit\"\n"))
		b.WriteString(fmt.Sprintf("          cwd \"%s\"\n", absPath))
		b.WriteString("        }\n")
	}
	b.WriteString("      }\n")

	// Right column: shell in each worktree
	b.WriteString("      pane split_direction=\"vertical\" size=\"60%\" {\n")
	for _, e := range entries {
		absPath, _ := filepath.Abs(e.Path)
		b.WriteString(fmt.Sprintf("        pane name=\"%s\" {\n", e.Slug))
		b.WriteString(fmt.Sprintf("          cwd \"%s\"\n", absPath))
		b.WriteString("        }\n")
	}
	b.WriteString("      }\n")

	b.WriteString("    }\n")
	b.WriteString("  }\n")

	// Status bar tab
	b.WriteString("  tab name=\"Manifest\" {\n")
	b.WriteString("    pane name=\"mochi status\" {\n")
	b.WriteString("      command \"watch\"\n")
	b.WriteString("      args \"-n\" \"2\" \"cat\" \".mochi_manifest.json\"\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")

	b.WriteString("}\n")
	return b.String()
}
