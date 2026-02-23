# MOCHI Examples

This directory contains example task files for MOCHI. Each file represents a single task -- the entire file content is sent as context to the AI agent.

## Quick Start

```bash
# Preview what would happen (no agents invoked)
mochi --input examples/add-user-auth.md --dry-run

# Run a task with the default model (Gemini 2.5 Pro)
mochi --input examples/add-user-auth.md

# Run with a specific model
mochi --input examples/add-user-auth.md --model claude-opus-4-6
```

## Example Files

| File | Type | Description |
|------|------|-------------|
| `add-user-auth.md` | Feature | JWT authentication system |
| `fix-mobile-navbar.md` | Bug fix | Mobile nav issues with repro steps |
| `api-test-suite.md` | Tests | Integration test suite for REST API |
| `database-migration.md` | Migration | SQLite to PostgreSQL migration |
| `PRD.md` | Multi-task | Sprint tasks (content sent as-is to agent) |
| `ISSUES.md` | Multi-task | Bug backlog (content sent as-is to agent) |

## Common Workflows

### Single task from a file

```bash
mochi --input examples/add-user-auth.md --model claude-opus-4-6 --create-prs
```

### Dry-run preview

```bash
mochi --input examples/fix-mobile-navbar.md --dry-run
```

### Concurrency control

```bash
# Limit to 2 concurrent worktrees
mochi --input examples/add-user-auth.md --worktrees 2
```

### Sequential debugging

```bash
mochi --input examples/fix-mobile-navbar.md --sequential --verbose
```

### Reviewer loop

```bash
# Use a reviewer model to iterate on the task
mochi --input examples/database-migration.md \
  --reviewer-model claude-opus-4-6 \
  --max-iterations 3 \
  --dry-run
```

### Output modes

```bash
# Write a research report instead of creating a PR
mochi --input examples/api-test-suite.md \
  --output-mode research-report \
  --output-dir ./reports

# Write raw output to a file
mochi --input examples/add-user-auth.md \
  --output-mode file \
  --output-dir ./output
```

### Worktree management

```bash
# List all tracked worktrees
mochi worktree list

# Check disk status of worktrees
mochi worktree status

# Remove a specific worktree
mochi worktree remove fix-mobile-navbar

# Remove ALL tracked worktrees
mochi worktree clean
```

### Model picker

```bash
# Interactively select a model before running
mochi --input examples/add-user-auth.md --prompt-model
```
