# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**MOCHI** (Multi-task AI Coding Orchestrator) — a Go CLI tool that runs AI agents (Claude, Gemini) in parallel across isolated git worktrees to execute coding tasks from any input source (file, or custom source plugins).

Binary name: `mochi` (historically also referenced as `ralphy`)

## Build & Run

```bash
# Build
go build -o mochi .

# Dry-run preview (no agents invoked)
./mochi --input examples/PRD.md --dry-run

# Run a task from any file (entire content becomes context)
./mochi --input examples/PRD.md

# Debug a single task sequentially
./mochi --input examples/PRD.md --task <slug> --sequential --verbose

# Override model for all tasks
./mochi --input examples/PRD.md --model claude-opus-4-6
```

## Tests & Linting

```bash
go test ./...                   # all tests
go test ./internal/source/...   # single package
```

No linter is configured. Standard Go formatting applies:
```bash
gofmt -w .
go vet ./...
```

## Architecture

The execution flow uses a plugin-based source system:

```
Source (file, or custom plugin: MCP, API, etc.)
    → source.Source interface → FetchTasks() → []source.Task
    → [orchestrator] creates git worktrees via [worktree.Manager]
    → parallel goroutines: [agent.Invoke] runs claude/gemini CLI
    → [github] pushes branches and opens PRs (if --create-prs)
    → colored summary, exit 1 on any failure
```

### Key Packages

- **`cmd/root.go`** — Cobra CLI, all flag definitions, worktree subcommands (`list`, `status`, `remove`, `clean`), routes to `orchestrator.Run(cfg)`
- **`internal/config/config.go`** — `Config` struct; defaults from `config/defaults.env`
- **`internal/source/source.go`** — `Source` interface + `Task` struct + `ToSlug` helper
- **`internal/source/file.go`** — `FileSource`: reads any file, uses filename as title, full content as description (one task)
- **`internal/worktree/worktree.go`** — Wraps `git worktree add`; persists state to `.mochi_manifest.json` (mutex-protected); handles branch collision with `-2`, `-3` suffixes
- **`internal/agent/agent.go`** — Routes to `claude --dangerously-skip-permissions -p <prompt>` or `gemini --model <model> -p <prompt>` based on model name prefix; writes logs to `logs/<slug>.log`; context timeout enforced
- **`internal/orchestrator/orchestrator.go`** — Main loop: resolve source → fetch tasks → create worktrees → run agents (parallel via `sync.WaitGroup` or `--sequential`, limited by `--worktrees N`) → PRs → cleanup
- **`internal/orchestrator/helpers.go`** — Task filtering (`filterBySlug`), slug formatting, status helpers
- **`internal/orchestrator/display.go`** — Terminal output styles (Lipgloss), dry-run display, summary printing
- **`internal/github/github.go`** — Shells out to `git push` and `gh pr create`
- **`internal/tui/model_picker.go`** — Interactive Bubble Tea model selector, triggered by `--prompt-model`
- **`internal/tui/progress.go`** — Progress TUI with spinner, live agent output preview, and summary
### Source Interface

New sources can be added by implementing the `Source` interface:

```go
type Source interface {
    Name() string
    FetchTasks() ([]Task, error)
}
```

Built-in source: `FileSource`. See `docs/ADDING_SOURCES.md` for adding custom sources.

### Provider Detection

Model prefix determines CLI tool invoked (`agent.go:buildCommand`):
- `gemini-*` → `gemini` CLI
- everything else → `claude` CLI

### Input Sources

Any file can be used as input — the entire content becomes the task context (no markdown pattern matching). The filename (without extension) becomes the task title.

Default model: `gemini-2.5-pro` (override via `MOCHI_MODEL` env var or `--model` flag).

## Runtime Artifacts

These are generated at runtime and gitignored:
- `.worktrees/` — git worktree copies of the repo
- `.mochi_manifest.json` — live state of all worktrees
- `logs/` — per-task agent output logs

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/bubbletea` — TUI framework (Bubble Tea)
- Requires `git`, `claude` (Claude Code CLI), and/or `gemini` CLIs to be installed and on PATH
- GitHub integration (`--create-prs`) requires `gh` CLI authenticated
