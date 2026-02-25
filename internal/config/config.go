package config

import "os"

// Config holds all runtime configuration for a MOCHI run.
type Config struct {
	// Input source
	InputFile   string
	IssueNumber int

	// Execution
	Model         string
	Timeout       int
	Sequential    bool
	TaskFilter    string
	DryRun        bool
	Verbose       bool
	KeepWorktrees bool
	CreatePRs     bool
	PromptModel   bool // show interactive model picker at startup
	MaxWorktrees  int  // max concurrent worktrees (0 = unlimited)

	// Git
	BaseBranch   string
	BranchPrefix string
	WorktreeDir  string

	// Output
	LogDir string

	// Ralph Loop
	ReviewerModel string // empty = no reviewer / no loop
	MaxIterations int    // default: 1 (single pass, no loop)
	OutputMode    string // pr | research-report | audit | knowledge-base | issue | file
	OutputDir     string // directory for file/report outputs

	// Workspace
	Workspace string // ai-native-dev workspace mode: "" (disabled), "zellij", "auto"
}

// Default returns a Config with sensible defaults.
// The default model can be overridden via the MOCHI_MODEL environment variable.
func Default() Config {
	model := "claude-sonnet-4-6"
	if env := os.Getenv("MOCHI_MODEL"); env != "" {
		model = env
	}

	return Config{
		Model:         model,
		InputFile:     "PRD.md",
		BaseBranch:    "main",
		BranchPrefix:  "feature",
		WorktreeDir:   ".worktrees",
		LogDir:        "logs",
		Timeout:       300000000,
		MaxIterations: 1,
		MaxWorktrees:  0,
		OutputMode:    "pr",
		OutputDir:     "output",
	}
}
