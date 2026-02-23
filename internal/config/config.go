package config

// Config holds all runtime configuration for a RALPHY run.
type Config struct {
	// Input source
	PRDFile     string
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
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Model:        "claude-sonnet-4-6",
		PRDFile:      "PRD.md",
		BaseBranch:   "main",
		BranchPrefix: "feature",
		WorktreeDir:  ".worktrees",
		LogDir:       "logs",
		Timeout:      300,
	}
}
