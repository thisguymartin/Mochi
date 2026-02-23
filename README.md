# MOCHI — Multi-Task AI Coding Orchestrator

> From any source → git worktrees → AI agents → PRs

MOCHI reads a task source (a file or any custom source plugin), spins up an **isolated git worktree per task**, invokes an AI agent in each worktree **in parallel**, and optionally opens GitHub pull requests when agents finish.

---

## Installation 

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/thisguymartin/Mochi/main/install.sh | sh
```

Or with `wget`:

```bash
wget -qO- https://raw.githubusercontent.com/thisguymartin/Mochi/main/install.sh | sh
```

The script will:
1. Detect your OS and architecture (macOS/Linux, amd64/arm64)
2. Download the latest pre-built binary from GitHub Releases
3. Fall back to building from source if no binary is available
4. Install `mochi` to `/usr/local/bin`

### Build from source

```bash
git clone https://github.com/thisguymartin/Mochi.git
cd Mochi
make build
```

The binary will be at `bin/mochi`. You can move it to your PATH:

```bash
sudo mv bin/mochi /usr/local/bin/
```

### Uninstall

```bash
sudo rm /usr/local/bin/mochi
```

---

## Quick Start

**Requirement:** The repo must have at least one commit (so the base branch, e.g. `main`, exists). If you see `fatal: invalid reference: main`, run:

```bash
git commit --allow-empty -m "Initial commit"
```

```bash
# Run with a file (entire content becomes the task context)
mochi --input examples/PRD.md

# Preview what would happen (no agents invoked)
mochi --input examples/PRD.md --dry-run
```

---

## Input Sources

MOCHI uses a plugin-based **Source** interface. Any file can be used as input — the entire content becomes the task context sent to the AI agent. Custom sources (MCP, API, etc.) can be added by implementing the `Source` interface. See [docs/ADDING_SOURCES.md](docs/ADDING_SOURCES.md) for a guide.

### File source (default)

Pass any file with `--input`. The filename (without extension) becomes the task title and branch slug. The full file content becomes the task description.

```bash
# Markdown spec
mochi --input architecture-spec.md

# Plain text TODO
mochi --input TODO.txt

# YAML, JSON — whatever your model can parse
mochi --input tasks.yaml
```

### Auto-detection

If `--input` defaults to `PRD.md` and that file doesn't exist, MOCHI searches for common alternatives: `PLAN.md`, `plan.md`, `input.md`, `tasks.md`, `docs/PLAN.md`, `docs/PRD.md`, `examples/PRD.md`.

---

## Commands & Flags

### Commands

| Command | Alias | Description |
|---|---|---|
| `mochi` | | Run tasks from a source |
| `mochi prune` | | Remove stale worktree registrations and manifest entries |
| `mochi worktree list` | `wt ls` | List all tracked worktrees |
| `mochi worktree status` | `wt status` | Show worktrees with disk-existence check |
| `mochi worktree remove <slug...>` | `wt rm` | Remove specific worktrees by slug |
| `mochi worktree clean` | `wt nuke` | Remove ALL tracked worktrees |

### Flags

| Flag | Default | Description |
|---|---|---|
| `--input <file>` / `-i` | `PRD.md` | Task file to read (any format). Aliases: `--plan`, `--prd`. |
| `--model <model-id>` | `gemini-2.5-pro` | Default AI model. Override via `MOCHI_MODEL` env var. |
| `--prompt-model` | `false` | Show interactive TUI model picker before running |
| `--worktrees <N>` | `0` (unlimited) | Max concurrent worktrees |
| `--reviewer-model <model-id>` | — | Model for the reviewer agent (enables the Ralph Loop) |
| `--max-iterations <num>` | `1` | Maximum worker iterations per task |
| `--output-mode <mode>` | `pr` | Output mode: `pr` \| `research-report` \| `audit` \| `knowledge-base` \| `issue` \| `file` |
| `--output-dir <dir>` | `output` | Directory for file/report outputs |
| `--create-prs` | `false` | Push branches and open GitHub PRs |
| `--dry-run` | `false` | Preview the plan without executing |
| `--sequential` | `false` | Run tasks one at a time (debug mode) |
| `--task <slug>` | — | Run only the task matching this slug |
| `--timeout <seconds>` | `3600` | Max time per agent |
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
| `claude-sonnet-4-6` | General purpose | Balanced |
| `claude-haiku-4-5` | Tests, docs, simple fixes | Lowest |

**Gemini** (requires `gemini` CLI)

| Model ID | Use Case |
|---|---|
| `gemini-2.5-pro` | Complex reasoning, large context tasks (default) |
| `gemini-2.0-flash` | Fast, cost-effective general purpose |
| `gemini-1.5-pro` | Long context, multimodal tasks |

---

## CLI Under the Hood & MCP Support

MOCHI delegates all AI work to the `claude` and `gemini` CLI tools. It does not call model APIs directly — it shells out to whichever CLI matches the model prefix.

This means:

- **MCP servers** configured for the `claude` CLI (in `~/.claude/settings.json` or project-level `.claude/settings.json`) are automatically available to agents spawned by MOCHI. No extra configuration is needed.
- **Claude CLI features** like `claude team agents`, tool permissions, and CLAUDE.md project instructions all work as expected, since MOCHI simply invokes `claude -p <prompt>` in each worktree.
- **Gemini CLI** configuration (API keys, settings) is similarly inherited by agents running under `gemini-*` models.

If you want agents to have access to databases, APIs, or other tools via MCP, configure them in your Claude/Gemini CLI settings and they will be available in every MOCHI task.

---

## Example Workflows

### Run a task from any file
```bash
mochi --input examples/add-user-auth.md --model claude-opus-4-6 --create-prs
```

### Debug a single task
```bash
mochi --input examples/PRD.md --task fix-mobile-navbar --sequential --verbose
```

### Interactive model picker
```bash
mochi --input PLAN.md --prompt-model
```

### Limit concurrent worktrees
```bash
mochi --input PLAN.md --worktrees 2
```

### Reviewer loop (Ralph Loop)
```bash
mochi --input examples/database-migration.md \
  --reviewer-model claude-opus-4-6 \
  --max-iterations 3
```

### Research report output
```bash
mochi --input examples/api-test-suite.md \
  --output-mode research-report \
  --output-dir ./reports
```

### Worktree management
```bash
# List all tracked worktrees
mochi worktree list

# Check disk status
mochi worktree status

# Remove a specific worktree
mochi worktree remove fix-mobile-navbar

# Remove ALL tracked worktrees
mochi worktree clean

# Clean up stale registrations after a crash
mochi prune
```

---

## What MOCHI Creates

During a run with a task:

```
.worktrees/
└── add-user-auth/            <- full repo copy on branch feature/add-user-auth

logs/
└── add-user-auth.log         <- full agent session output + timestamps

.mochi_manifest.json          <- live task status tracking
```

Worktrees and the manifest are cleaned up at the end of each run unless `--keep-worktrees` is set.

---

## Architecture

```
Source (file, or custom plugin: MCP, API, etc.)
    -> source.Source interface -> FetchTasks() -> []source.Task
    -> [orchestrator] creates git worktrees via [worktree.Manager]
    -> parallel goroutines: [agent.Invoke] runs claude/gemini CLI
    -> [github] pushes branches and opens PRs (if --create-prs)
    -> colored summary, exit 1 on any failure
```

### Source Interface

New sources can be added by implementing:

```go
type Source interface {
    Name() string
    FetchTasks() ([]Task, error)
}
```

Built-in source: `FileSource` (any file). See [docs/ADDING_SOURCES.md](docs/ADDING_SOURCES.md) to add custom sources.

### Provider Detection

Model prefix determines which CLI tool is invoked:
- `gemini-*` -> `gemini` CLI
- everything else -> `claude` CLI

---

## Project Structure

```text
mochi/
├── main.go                              # Entry point
├── cmd/
│   └── root.go                          # CLI flags + worktree subcommands (Cobra)
├── internal/
│   ├── agent/agent.go                   # AI CLI invocation (Claude/Gemini)
│   ├── config/config.go                 # Config struct and defaults
│   ├── github/github.go                 # git push + gh pr create
│   ├── memory/memory.go                 # Ralph Loop persistence
│   ├── orchestrator/
│   │   ├── orchestrator.go              # Main run loop + ResolveSource
│   │   ├── helpers.go                   # Task filtering, slug formatting
│   │   └── display.go                   # Terminal output (Lipgloss styles)
│   ├── output/output.go                 # Output dispatch (PRs, files, reports)
│   ├── reviewer/reviewer.go             # Ralph Loop reviewer logic
│   ├── source/
│   │   ├── source.go                    # Source interface + Task struct + ToSlug
│   │   └── file.go                      # FileSource: any file -> single task
│   ├── tui/                             # Terminal UI (splash, model picker, progress)
│   └── worktree/worktree.go             # Git worktree manager + manifest
├── config/defaults.env                  # Default values reference
├── examples/                            # Example task files
└── logs/                                # Agent log output (gitignored)
```

---

## Requirements

- Go 1.22+
- `git` (for worktree management)
- `claude` CLI — [Claude Code](https://claude.ai/code) — required for `claude-*` models
- `gemini` CLI — [Gemini CLI](https://github.com/google-gemini/gemini-cli) — required for `gemini-*` models
- `gh` CLI — only required for `--create-prs`

---

## Roadmap

Planned features and improvements — roughly ordered by priority.

### Source Plugins
- [ ] **GitHub Issue source** — fetch task context from a GitHub Issue via `gh` CLI (`--issue <number>`)
- [ ] **Linear source** — pull tasks from Linear tickets (`--linear <issue-id>`)
- [ ] **MCP source** — fetch tasks by calling an MCP server tool (`--mcp-server <name> --mcp-tool <tool>`)
- [ ] **REST API source** — generic HTTP endpoint for task ingestion (`--api-endpoint <url>`)
- [ ] **Jira source** — pull tasks from Jira tickets
- [ ] **Notion source** — read task specs from Notion pages/databases

### Agent & Orchestration
- [ ] **Multi-task file parsing** — parse a single input file into multiple independent tasks (e.g. checklist items, YAML task lists)
- [ ] **Task dependency graph** — define dependencies between tasks so they execute in the correct order
- [ ] **Agent-to-agent handoff** — allow one task's output to feed into another task's input
- [ ] **Retry with backoff** — automatically retry failed agent invocations with exponential backoff
- [ ] **Cost estimation** — estimate token usage and cost before running (shown in `--dry-run`)
- [ ] **Progress TUI** — real-time progress dashboard with per-task status, spinner, and live log tailing

### Output & Integration
- [ ] **Slack/Discord notifications** — post summaries to a channel when a run completes
- [ ] **Webhook output** — POST results to a configurable endpoint on completion
- [ ] **PR auto-review** — use the reviewer model to leave inline comments on the generated PR
- [ ] **Merge strategy** — optionally auto-merge PRs that pass CI (`--auto-merge`)
- [ ] **Custom PR templates** — configurable PR body templates with task metadata

### Developer Experience
- [ ] **`mochi init`** — scaffold a project with example task files and config
- [ ] **Config file support** — `.mochi.yaml` / `.mochi.toml` for per-project defaults (model, branch prefix, output mode, etc.)
- [ ] **Plugin registry** — discover and install community source plugins
- [ ] **`mochi logs`** — TUI for browsing and searching past run logs
- [ ] **`mochi replay`** — re-run a previous task with the same context

### Infrastructure
- [ ] **CI/CD integration guide** — documented patterns for running MOCHI in GitHub Actions, GitLab CI, etc.
- [ ] **Containerized runs** — Docker image for sandboxed agent execution
- [ ] **Telemetry & analytics** — opt-in run metrics (duration, success rate, iterations) for team dashboards
