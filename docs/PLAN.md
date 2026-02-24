# RALPHY Build Plan

> Last updated: 2026-02-23
> Language: Go 1.22
> Binary: `ralphy`

---

## Phase 1 — Project Foundation ✅

**What:** Skeleton of the project — git repo, directory structure, Go module, and default config.
**Why:** Every subsequent phase needs a stable home. Getting structure right first prevents scattered paths and hardcoded values.

- [x] Initialize git repository
- [x] Create directory structure (`cmd/`, `internal/`, `config/`, `logs/`)
- [x] Write `go.mod` with module path `github.com/thisguymartin/ai-forge`
- [x] Write `main.go` entry point
- [x] Write `config/defaults.env` with sensible defaults
- [x] Write `internal/config/config.go` — `Config` struct and `Default()` constructor

---

## Phase 2 — Task File Parser ✅

**What:** Reads a markdown file with a `## Tasks` section, parses bullet points into normalized `Task` structs with optional per-task model annotations.
**Why:** Everything downstream depends on a clean task list. The same parser handles both local files and GitHub Issue bodies fetched in Phase 6.

- [x] Define task file format (`PRD.md`, `ISSUES.md`)
- [x] Write `internal/parser/parser.go`
  - [x] Read file line by line with `bufio.Scanner`
  - [x] Detect `## Tasks` section, ignore everything outside it
  - [x] Strip bullet prefixes (`- `, `* `)
  - [x] Skip blank lines and `#` comments
  - [x] Extract optional `[model:...]` annotation per task
  - [x] Convert description to branch-safe slug via `toSlug()`
- [x] Return meaningful error when no tasks are found
- [x] Write example `PRD.md` and `ISSUES.md`

---

## Phase 3 — Git Worktree Manager ✅

**What:** Creates an isolated git worktree per task on its own branch so agents can work in parallel without conflicts.
**Why:** True parallel execution requires isolation. Each agent gets a full copy of the repo. Worktrees are tracked in a JSON manifest and cleaned up after the run.

- [x] Write `internal/worktree/worktree.go`
  - [x] `NewManager()` — configures root, base branch, prefix, and worktree dir
  - [x] `Create(slug)` — runs `git worktree add -b <branch> <path> <base>`
  - [x] `Destroy(slug)` — removes worktree and deletes branch
  - [x] `UpdateStatus(slug, status)` — updates manifest entry
  - [x] `GetEntry(slug)` — retrieves entry from manifest
  - [x] `resolveBranch()` — appends `-2`, `-3` suffix on collision
  - [x] JSON manifest file (`.ralphy_manifest.json`) with mutex for concurrent writes
  - [x] `MkdirAll` for worktree directory creation

---

## Phase 4 — AI Agent Invocation ✅

**What:** Invokes the `claude` CLI inside each worktree with a focused, context-rich prompt. Logs all output to per-task files. Supports model routing and timeouts.
**Why:** The quality of output depends on the prompt and model. This layer encapsulates prompt construction and Claude CLI invocation so the orchestrator stays clean.

- [x] Write `internal/agent/agent.go`
  - [x] `Invoke(opts, slug)` — builds prompt, runs `claude --dangerously-skip-permissions -p <prompt>`
  - [x] `buildPrompt()` — uses `text/template` with worktree path, branch name, and task description
  - [x] `detectBranch()` — reads current branch name from worktree via `git branch --show-current`
  - [x] `context.WithTimeout` — kills agent after configured timeout
  - [x] Log all agent stdout/stderr to `logs/<slug>.log` with timestamps
  - [x] `--verbose` support — tees agent output to terminal via `io.MultiWriter`
  - [x] Returns `Result` struct: slug, success bool, duration, log path, error

---

## Phase 5 — RALPHY Orchestration Loop ✅

**What:** The main `orchestrator.Run()` function that ties all phases together — parses tasks, creates worktrees, fires agents in parallel goroutines, collects results, and prints a summary.
**Why:** This is what users actually invoke. A clean loop means fast parallel runs, clear failure reporting, and easy debugging modes.

- [x] Write `internal/orchestrator/orchestrator.go`
  - [x] Resolve task source (file or GitHub Issue)
  - [x] Parse tasks and apply default model
  - [x] Apply `--task` filter for single-task runs
  - [x] `--dry-run` mode — prints plan without executing
  - [x] Create all worktrees before starting agents
  - [x] Parallel mode — `sync.WaitGroup` + goroutines, one per task
  - [x] Sequential mode — simple loop for debugging
  - [x] Collect `agent.Result` for each task
  - [x] Color-coded terminal output (✓ green / ✗ red / ⚠ yellow)
  - [x] Final summary line: `N succeeded, N failed`
  - [x] `os.Exit(1)` if any task failed (CI-compatible)
  - [x] Cleanup worktrees unless `--keep-worktrees`
- [x] Write `cmd/root.go` — all CLI flags via cobra
- [x] Wire everything through `main.go`

---

## Phase 6 — GitHub MCP Integration

**What:** Pushes completed branches to GitHub and opens pull requests via the `gh` CLI. Also enables pulling tasks directly from live GitHub Issues.
**Why:** Without this, RALPHY produces local branches only. With it, a single command takes tasks all the way to open PRs.

- [x] Write `internal/github/github.go`
  - [x] `PushBranch(repoRoot, branch)` — `git push -u origin <branch>`
  - [x] `CreatePR(opts)` — `gh pr create` with title, body, and `ralphy-generated` label
  - [x] `FetchIssueTasks(issueNumber, repoRoot)` — `gh issue view --json body` → temp file
  - [x] `buildPRBody()` — markdown body with task summary + last 20 lines of agent log
  - [x] `readLogSummary()` — reads tail of log for PR body
- [x] Wire `--create-prs` flag through orchestrator to `github.PushBranch` + `github.CreatePR`
- [x] Wire `--issue` flag through orchestrator to `github.FetchIssueTasks`
- [ ] Integration test: end-to-end with a real GitHub repo and issue

---

## Phase 7 — Polish & DX

**What:** Final layer — better error messages, dependency checks, timeout flag, README completeness.
**Why:** A tool that works but is painful to use gets abandoned. This phase makes RALPHY production-quality.

- [x] Color-coded terminal output (phases 1–5 already implemented)
- [x] `--help` auto-generated by cobra with examples
- [x] `README.md` with quick start, format spec, all flags, and example workflows
- [x] Startup dependency checks (warn if `claude`, `git`, or `gh` are not in `$PATH`)
- [ ] `--timeout` propagated correctly to agent context (already wired, needs e2e test)
- [ ] `go build` produces a single static binary `ralphy`
- [x] `go test ./...` — unit tests for parser, worktree slug resolution, PR body builder

---

## Build & Run

```bash
# Install dependencies
go mod tidy

# Build binary
go build -o ralphy .

# Run dry-run to verify setup
./ralphy --prd PRD.md --dry-run
```

---

## Milestone Summary

| Phase | Deliverable | Status |
|---|---|---|
| 1 | Project structure + Go module | ✅ Done |
| 2 | Task file parser | ✅ Done |
| 3 | Git worktree manager | ✅ Done |
| 4 | AI agent invocation | ✅ Done |
| 5 | Orchestration loop + CLI | ✅ Done |
| 6 | GitHub MCP + PRs | ✅ Done |
| 7 | Polish + tests + DX | 🔄 In Progress |
