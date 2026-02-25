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

MOCHI accepts **any text file** as input — markdown, plain text, YAML, or anything your AI model can understand. It uses multi-strategy task detection:

### Strategy 1: Structured task sections (highest priority)

Tasks under recognized headings (`## Tasks`, `## Todo`, `## Action Items`, `## Steps`, `## Checklist`):

```markdown
## Tasks
- Add user auth
- Fix mobile navbar
- Add dark mode [model:claude-opus-4-6]
- Write API tests [model:claude-haiku-4-5]
```

Supports bullets (`-`, `*`), numbered lists (`1.`, `2)`), and checkboxes:

```markdown
## Todo
- [ ] Add user authentication
- [x] Fix mobile navbar (skipped — already done)
- [ ] Write API tests [model:claude-haiku-4-5]

1. Refactor auth module
2. Add rate limiting [model:gemini-2.5-pro]
```

### Strategy 2: Checkboxes anywhere

If no task section is found, MOCHI scans the entire file for markdown checkboxes. Completed `[x]` items are skipped:

```markdown
# Sprint Plan

Some context about the project...

- [ ] First implement the API
- [x] Already set up the database
- [ ] Write integration tests
```

### Strategy 3: Whole-file fallback

If no tasks or checkboxes are detected, the entire file becomes a single task — perfect for feeding a plan document or spec directly to an AI agent:

```bash
mochi --input architecture-spec.md
mochi --input TODO.txt
```

**Annotations** (work in any strategy):
- `[model:<model-id>]` — per-task model override
- `[title:<name>]` — explicit short title for the branch name

---

## Commands & Flags

### Commands

- `mochi prune`: Remove stale worktree registrations and manifest entries.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--input <file>` | `PRD.md` | Task file to read (any text format). Aliases: `--plan`, `--prd`. Auto-detects `PLAN.md`, `input.md`, etc. if default missing. |
| `--issue <number>` | — | Pull tasks from a GitHub Issue |
| `--model <model-id>` | `claude-sonnet-4-6` | Default Claude or Gemini model. Override via `MOCHI_MODEL` env var. |
| `--prompt-model` | `false` | Show interactive TUI model picker before running |
| `--worktrees <N>` | `0` (unlimited) | Max concurrent worktrees. Useful for resource-constrained machines. |
| `--reviewer-model <model-id>` | — | Model for the reviewer agent (enables the Ralph Loop) |
| `--max-iterations <num>` | `1` | Maximum worker iterations per task |
| `--output-mode <mode>` | `pr` | Output mode: pr \| research-report \| audit \| knowledge-base \| issue \| file |
| `--output-dir <dir>` | `output` | Directory for file/report outputs |
| `--create-prs` | `false` | Push branches and open GitHub PRs |
| `--dry-run` | `false` | Preview the plan without executing |
| `--sequential` | `false` | Run tasks one at a time (debug mode) |
| `--task <slug>` | — | Run only the task matching this slug |
| `--timeout <seconds>` | `3000` | Max time per agent |
| `--verbose` | `false` | Stream agent output live to terminal |
| `--keep-worktrees` | `false` | Keep worktrees on disk after run |
| `--base-branch <branch>` | `main` | Branch to base worktrees on |
| `--workspace <mode>` | — | Launch ai-native-dev workspace with worktree panes (`zellij` \| `auto`) |

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

### Interactive model picker

```bash
./mochi --input PLAN.md --prompt-model
```
Shows a TUI model selector before running. Navigate with arrow keys or j/k, Enter to select.

### Limit concurrent worktrees

```bash
./mochi --input PLAN.md --worktrees 2
```
If the plan has 5 tasks, only 2 agents run at a time. The rest queue up.

### Launch workspace with live worktree view

```bash
./mochi --input PLAN.md --workspace zellij --keep-worktrees
```
Creates worktrees and launches a Zellij session with one pane per worktree. Left column shows LazyGit for each branch, right column gives you a shell inside each worktree. A "Manifest" tab auto-refreshes `.mochi_manifest.json` for live status.

### Feed any file as a plan

```bash
# Plain text
mochi --input TODO.txt

# A spec document — becomes a single task
mochi --input architecture-spec.md

# YAML, JSON — whatever your model can parse
mochi --input tasks.yaml
```

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
- `zellij` — only required for `--workspace zellij` ([zellij.dev](https://zellij.dev))
- `lazygit` — optional, used in workspace panes for git visualization

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
│   ├── parser/parser.go            # Multi-strategy task file parser
│   ├── reviewer/reviewer.go        # Ralph Loop reviewer logic
│   ├── tui/                        # Terminal UI (splash, model picker)
│   ├── workspace/workspace.go      # ai-native-dev / Zellij integration
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
