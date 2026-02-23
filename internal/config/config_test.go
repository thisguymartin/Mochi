package config

import (
	"os"
	"testing"
)

func TestDefault_Model(t *testing.T) {
	// Unset env to test true default
	orig := os.Getenv("MOCHI_MODEL")
	os.Unsetenv("MOCHI_MODEL")
	defer os.Setenv("MOCHI_MODEL", orig)

	cfg := Default()
	if cfg.Model != "gemini-2.5-pro" {
		t.Errorf("Default model = %q; want %q", cfg.Model, "gemini-2.5-pro")
	}
}

func TestDefault_ModelEnvOverride(t *testing.T) {
	orig := os.Getenv("MOCHI_MODEL")
	os.Setenv("MOCHI_MODEL", "claude-opus-4-6")
	defer func() {
		if orig == "" {
			os.Unsetenv("MOCHI_MODEL")
		} else {
			os.Setenv("MOCHI_MODEL", orig)
		}
	}()

	cfg := Default()
	if cfg.Model != "claude-opus-4-6" {
		t.Errorf("Default model with env = %q; want %q", cfg.Model, "claude-opus-4-6")
	}
}

func TestDefault_BaseBranch(t *testing.T) {
	orig := os.Getenv("MOCHI_MODEL")
	os.Unsetenv("MOCHI_MODEL")
	defer os.Setenv("MOCHI_MODEL", orig)

	cfg := Default()
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q; want %q", cfg.BaseBranch, "main")
	}
}

func TestDefault_WorktreeDir(t *testing.T) {
	orig := os.Getenv("MOCHI_MODEL")
	os.Unsetenv("MOCHI_MODEL")
	defer os.Setenv("MOCHI_MODEL", orig)

	cfg := Default()
	if cfg.WorktreeDir != ".worktrees" {
		t.Errorf("WorktreeDir = %q; want %q", cfg.WorktreeDir, ".worktrees")
	}
}

func TestDefault_OutputMode(t *testing.T) {
	orig := os.Getenv("MOCHI_MODEL")
	os.Unsetenv("MOCHI_MODEL")
	defer os.Setenv("MOCHI_MODEL", orig)

	cfg := Default()
	if cfg.OutputMode != "pr" {
		t.Errorf("OutputMode = %q; want %q", cfg.OutputMode, "pr")
	}
}
