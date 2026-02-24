# MOCHI — Multi-Task AI Coding Orchestrator

> From task file → git worktrees → AI agents → PRs

MOCHI reads a markdown task list, spins up an **isolated git worktree per task**, invokes an AI agent in each worktree **in parallel**, and optionally opens GitHub pull requests when agents finish.

---

## Quick Start

**Requirement:** The repo must have at least one commit (so the base branch, e.g. `main`, exists). If you see `fatal: invalid reference: main`, run:

```bash
git commit --allow-empty -m "Initial commit"
```

```bash
# Build
make build

# Run (auto-detects PRD.md or PLAN.md)
./mochi

# Run with a specific plan
./mochi --input docs/PLAN.md --dry-run

# Run and auto-open PRs
./mochi --create-prs
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

## Commands & Flags

### Commands

- `mochi prune`: Remove stale worktree registrations and manifest entries.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--input <file>` | `PRD.md` | Task file to read. Aliases: `--plan`, `--prd`. Auto-detects `PLAN.md`, `input.md`, etc. if default missing. |
| `--issue <number>` | — | Pull tasks from a GitHub Issue |
| `--model <model-id>` | `claude-sonnet-4-6` | Default Claude or Gemini model |
| `--reviewer-model <model-id>` | — | Model for the reviewer agent (enables the Ralph Loop) |
| `--max-iterations <num>` | `1` | Maximum worker iterations per task |
| `--output-mode <mode>` | `pr` | Output mode: pr \| research-report \| audit \| knowledge-base \| issue \| file |
| `--output-dir <dir>` | `output` | Directory for file/report outputs |
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

### Sprint execution from task file
```bash
./mochi --create-prs
```
Auto-detects `PRD.md` or `PLAN.md`, spins up one worktree per task in parallel, runs agents, opens a PR per completed task.

### Pull tasks from a GitHub Issue
```bash
./mochi --issue 88 --create-prs
```
Fetches Issue #88 body via `gh`, parses the `## Tasks` section, runs agents, opens PRs.

### Debug a single failing task
```bash
./mochi --input examples/PRD.md --task fix-mobile-navbar --sequential --verbose
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
./mochi --input examples/PRD.md
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

```text
mochi/
├── main.go                         # Entry point
├── cmd/
│   └── root.go                     # CLI flags via cobra
├── internal/
│   ├── agent/agent.go              # AI CLI invocation (Claude/Gemini)
│   ├── config/config.go            # Config struct and defaults
│   ├── github/github.go            # GitHub PR + Issue integration
│   ├── memory/memory.go            # Ralph Loop persistence
│   ├── orchestrator/orchestrator.go # Main run loop
│   ├── output/output.go            # Output dispatch (PRs, files, etc)
│   ├── parser/parser.go            # Task file parser
│   ├── reviewer/reviewer.go        # Ralph Loop reviewer logic
│   ├── tui/                        # Terminal UI components
│   └── worktree/worktree.go        # Git worktree manager
├── config/defaults.env             # Default values reference
├── docs/                           # Documentation and architecture
├── examples/                       # Example sprint/issue task files
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
