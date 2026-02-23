# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**MOCHI** (Multi-task AI Coding Orchestrator) — a Go CLI tool that runs AI agents (Claude, Gemini) in parallel across isolated git worktrees to execute coding tasks from a markdown PRD file or GitHub Issue.

Binary name: `mochi` (historically also referenced as `ralphy`)

## Build & Run

```bash
# Build
go build -o mochi .

# Dry-run preview (no agents invoked)
./mochi --prd examples/PRD.md --dry-run

# Run all tasks in parallel
./mochi --prd examples/PRD.md

# Run tasks from a GitHub Issue
./mochi --issue 88 --create-prs

# Debug a single task sequentially
./mochi --prd examples/PRD.md --task <slug> --sequential --verbose

# Override model for all tasks
./mochi --prd examples/PRD.md --model claude-opus-4-6
```

## Tests & Linting

No test infrastructure exists yet (Phase 7 is pending). When adding tests:

```bash
go test ./...
go test ./internal/parser/...   # single package
```

No linter is configured. Standard Go formatting applies:
```bash
gofmt -w .
go vet ./...
```

## Architecture

The execution flow is:

```
Task Source (PRD.md or GitHub Issue)
    → [parser] parses markdown → []Task{Description, Slug, Model}
    → [orchestrator] creates git worktrees via [worktree.Manager]
    → parallel goroutines: [agent.Invoke] runs claude/gemini CLI
    → [github] pushes branches and opens PRs (if --create-prs)
    → colored summary, exit 1 on any failure
```

### Key Packages

- **`cmd/root.go`** — Cobra CLI, all flag definitions, routes to `orchestrator.Run(cfg)`
- **`internal/config/config.go`** — `Config` struct; defaults from `config/defaults.env`
- **`internal/parser/parser.go`** — Reads markdown, extracts `## Tasks` section; supports per-task `[model:...]` overrides; `toSlug()` creates branch-safe identifiers
- **`internal/worktree/worktree.go`** — Wraps `git worktree add`; persists state to `.mochi_manifest.json` (mutex-protected); handles branch collision with `-2`, `-3` suffixes
- **`internal/agent/agent.go`** — Routes to `claude --dangerously-skip-permissions -p <prompt>` or `gemini --model <model> -p <prompt>` based on model name prefix; writes logs to `logs/<slug>.log`; context timeout enforced
- **`internal/orchestrator/orchestrator.go`** — Main loop: parse → create worktrees → run agents (parallel via `sync.WaitGroup` or `--sequential`) → PRs → cleanup
- **`internal/github/github.go`** — Shells out to `git push` and `gh pr create`; fetches issue body via `gh issue view --json body`

### Provider Detection

Model prefix determines CLI tool invoked (`agent.go:buildCommand`):
- `gemini-*` → `gemini` CLI
- everything else → `claude` CLI

### Task File Format

Markdown files must have a `## Tasks` section with bullet-point tasks:
```markdown
## Tasks
- Add user authentication [model:claude-opus-4-6]
- Fix mobile navbar bug
```

## Runtime Artifacts

These are generated at runtime and gitignored:
- `.worktrees/` — git worktree copies of the repo
- `.mochi_manifest.json` — live state of all worktrees
- `logs/` — per-task agent output logs

## Dependencies

- `github.com/spf13/cobra` — only external dependency
- Requires `git`, `claude` (Claude Code CLI), and/or `gemini` CLIs to be installed and on PATH
- GitHub integration requires `gh` CLI authenticated
