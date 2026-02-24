package config

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
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Model:         "claude-sonnet-4-6",
		InputFile:     "PRD.md",
		BaseBranch:    "main",
		BranchPrefix:  "feature",
		WorktreeDir:   ".worktrees",
		LogDir:        "logs",
		Timeout:       300000000,
		MaxIterations: 1,
		OutputMode:    "pr",
		OutputDir:     "output",
	}
}
