package provider

import (
	"context"
	"os/exec"
	"strings"
)

// Tool describes an external binary required for a provider.
type Tool struct {
	Name    string
	Install string
}

// Provider holds provider metadata used for dispatch and dependency checks.
type Provider struct {
	Name string
	Tool Tool
}

var (
	claudeProvider = Provider{
		Name: "claude",
		Tool: Tool{Name: "claude", Install: "https://claude.ai/code"},
	}
	geminiProvider = Provider{
		Name: "gemini",
		Tool: Tool{Name: "gemini", Install: "https://ai.google.dev/gemini-api/docs/gemini-cli"},
	}
	codexProvider = Provider{
		Name: "codex",
		Tool: Tool{Name: "codex", Install: "https://developers.openai.com/codex/cli"},
	}
)

// Detect picks a provider based on model prefix.
//
// Rules:
// - gemini-* -> gemini
// - gpt-* / o1* / o3* / o4* / codex-* -> codex
// - everything else defaults to claude (backward compatible)
func Detect(model string) Provider {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "gemini-"):
		return geminiProvider
	case strings.HasPrefix(m, "gpt-"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "o4"),
		strings.HasPrefix(m, "codex-"):
		return codexProvider
	default:
		return claudeProvider
	}
}

// ToolForModel returns the required local CLI tool for a model.
func ToolForModel(model string) Tool {
	return Detect(model).Tool
}

// BuildCommand constructs the provider-specific command for non-interactive runs.
func BuildCommand(ctx context.Context, model, prompt string) *exec.Cmd {
	switch Detect(model).Name {
	case "gemini":
		return exec.CommandContext(ctx, "gemini", "--model", model, "-p", prompt)
	case "codex":
		return exec.CommandContext(ctx, "codex", "exec", "--model", model, prompt)
	default:
		return exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", prompt)
	}
}
