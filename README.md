# MOCHI — Multi-Task AI Coding Orchestrator

> From task file → git worktrees → AI agents → PRs

MOCHI reads a markdown task list, spins up an **isolated git worktree per task**, invokes a Claude AI agent in each worktree **in parallel**, and optionally opens GitHub pull requests when agents finish.

---

## Quick Start

**Requirement:** The repo must have at least one commit (so the base branch, e.g. `main`, exists). If you see `fatal: invalid reference: main`, run:

```bash
git commit --allow-empty -m "Initial commit"
```

```bash
# Build
go build -o mochi .

# Run with the sample PRD
./mochi --prd examples/PRD.md

# Dry-run first to preview
./mochi --prd examples/PRD.md --dry-run

# Run and auto-open PRs
./mochi --prd examples/PRD.md --create-prs
```

---

## Task File Format

Create a markdown file with a `## Tasks` section. Each bullet becomes one task:

```markdown
## Tasks
- Add user auth
- Fix mobile navbar
- Add dark mode [model:claude-opus-4-6]
- Write API tests [model:claude-haiku-4-5]
```

**Rules:**
- Tasks must be under a `## Tasks` heading
- Each task is a `- ` or `* ` bullet point
- Lines starting with `#` or blank lines are ignored
- Optionally annotate a per-task model with `[model:<model-id>]`

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--prd <file>` | `PRD.md` | Task file to read |
| `--issue <number>` | — | Pull tasks from a GitHub Issue |
| `--model <model-id>` | `claude-sonnet-4-6` | Default Claude model |
| `--create-prs` | `false` | Push branches and open GitHub PRs |
| `--dry-run` | `false` | Preview the plan without executing |
| `--sequential` | `false` | Run tasks one at a time (debug mode) |
| `--task <slug>` | — | Run only the task matching this slug |
| `--timeout <seconds>` | `300` | Max time per agent |
| `--verbose` | `false` | Stream agent output live to terminal |
| `--keep-worktrees` | `false` | Keep worktrees on disk after run |
| `--base-branch <branch>` | `main` | Branch to base worktrees on |

---

## Models

The provider is auto-detected from the model name prefix (`claude-*` or `gemini-*`).

**Claude** (requires `claude` CLI)

| Model ID | Use Case | Cost |
|---|---|---|
| `claude-opus-4-6` | Complex architecture, migrations, auth | Highest |
| `claude-sonnet-4-6` | General purpose (default) | Balanced |
| `claude-haiku-4-5` | Tests, docs, simple fixes | Lowest |

**Gemini** (requires `gemini` CLI)

| Model ID | Use Case |
|---|---|
| `gemini-2.5-pro` | Complex reasoning, large context tasks |
| `gemini-2.0-flash` | Fast, cost-effective general purpose |
| `gemini-1.5-pro` | Long context, multimodal tasks |

---

## Example Workflows

### Sprint execution from PRD
```bash
./mochi --prd examples/PRD.md --create-prs
```
Reads `examples/PRD.md`, spins up one worktree per task in parallel, runs agents, opens a PR per completed task.

### Pull tasks from a GitHub Issue
```bash
./mochi --issue 88 --create-prs
```
Fetches Issue #88 body via `gh`, parses the `## Tasks` section, runs agents, opens PRs.

### Debug a single failing task
```bash
./mochi --prd examples/PRD.md --task fix-mobile-navbar --sequential --verbose
```
Runs only `fix-mobile-navbar` sequentially with live output streamed to terminal.

### Cost-optimized mixed run
```markdown
## Tasks
- Refactor auth system [model:claude-opus-4-6]
- Fix typo in README [model:claude-haiku-4-5]
- Add pagination [model:claude-sonnet-4-6]
```
```bash
./mochi --prd examples/PRD.md
```
Each task uses its annotated model — Opus for the hard stuff, Haiku for the trivial.

---

## What MOCHI Creates

During a run with 3 tasks:

```
.worktrees/
├── fix-mobile-navbar/     ← full repo copy on branch feature/fix-mobile-navbar
├── add-dark-mode/         ← full repo copy on branch feature/add-dark-mode
└── write-api-tests/       ← full repo copy on branch feature/write-api-tests

logs/
├── fix-mobile-navbar.log  ← full agent session output + timestamps
├── add-dark-mode.log
└── write-api-tests.log

.mochi_manifest.json      ← live task status tracking
```

Worktrees and the manifest are cleaned up at the end of each run unless `--keep-worktrees` is set.

---

## Requirements

- Go 1.22+
- `git` (for worktree management)
- `claude` CLI — [Claude Code](https://claude.ai/code) — required for `claude-*` models
- `gemini` CLI — [Gemini CLI](https://github.com/google-gemini/gemini-cli) — required for `gemini-*` models
- `gh` CLI — only required for `--create-prs` and `--issue` flags

---

## Project Structure

```
ai-forge/
├── main.go                         # Entry point
├── cmd/
│   └── root.go                     # CLI flags via cobra
├── internal/
│   ├── config/config.go            # Config struct and defaults
│   ├── parser/parser.go            # Task file parser
│   ├── worktree/worktree.go        # Git worktree manager
│   ├── agent/agent.go              # Claude CLI invocation
│   ├── github/github.go            # GitHub PR + Issue integration
│   └── orchestrator/orchestrator.go # Main run loop
├── config/defaults.env             # Default values reference
├── docs/
│   └── PLAN.md                     # Build plan and todo checklist
├── examples/
│   ├── PRD.md                      # Example sprint task file
│   └── ISSUES.md                   # Example bug/issue task file
└── logs/                           # Agent log output
```

---

## How It Works

```
PRD.md / ISSUES.md / GitHub Issue
         │
         ▼
   [parser] reads tasks line by line
         │
         ▼
   [worktree manager] creates feature/<slug> branch + worktree per task
         │
         ▼ (parallel goroutines)
   [agent] invokes claude CLI inside each worktree with focused prompt
         │
         ▼
   [github] pushes branches, opens PRs (if --create-prs)
         │
         ▼
   Summary: N succeeded, N failed
```
