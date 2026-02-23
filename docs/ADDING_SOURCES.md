# Adding a New Source Plugin

MOCHI uses a plugin-based architecture for task sources. A **source** is anything that can produce one or more `Task` structs — a file, an API call, an MCP server, a database query, etc.

This guide walks through adding a new source from scratch.

---

## The Source Interface

Every source implements this interface (defined in `internal/source/source.go`):

```go
type Source interface {
    Name() string
    FetchTasks() ([]Task, error)
}
```

| Method | Purpose |
|---|---|
| `Name()` | Returns a short identifier for logging (e.g. `"mcp"`, `"linear"`, `"api"`) |
| `FetchTasks()` | Fetches and returns one or more tasks. This is where your integration logic lives. |

Each task is a simple struct:

```go
type Task struct {
    Title       string // Short title — becomes the branch name slug
    Description string // Full context sent to the AI agent
    Slug        string // Branch-safe identifier (use source.ToSlug(title))
    Model       string // Optional per-task model override (empty = use default)
}
```

---

## Step-by-Step

### 1. Create the source file

Add a new file in `internal/source/`. Name it after your integration:

```
internal/source/mcp.go        # for an MCP source
internal/source/linear.go     # for Linear
internal/source/webhook.go    # for a webhook/API source
```

### 2. Implement the Source interface

Here are two complete examples — one for an MCP server, one for a generic REST API.

#### Example: MCP Source

```go
package source

import (
    "encoding/json"
    "fmt"
    "os/exec"
)

// MCPSource fetches tasks from an MCP server tool call.
type MCPSource struct {
    ServerName string // MCP server identifier
    ToolName   string // MCP tool to call (e.g. "get_tasks")
    Args       string // JSON args to pass to the tool
}

func (s *MCPSource) Name() string { return "mcp" }

func (s *MCPSource) FetchTasks() ([]Task, error) {
    // Shell out to the claude CLI with MCP tool use, or call
    // the MCP server directly via its stdio transport.
    // This example assumes a helper that returns JSON.
    cmd := exec.Command("your-mcp-client",
        "--server", s.ServerName,
        "--tool", s.ToolName,
        "--args", s.Args,
    )
    out, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("mcp call failed: %w\n%s", err, string(out))
    }

    var items []struct {
        Title       string `json:"title"`
        Description string `json:"description"`
    }
    if err := json.Unmarshal(out, &items); err != nil {
        return nil, fmt.Errorf("mcp response parse error: %w", err)
    }

    tasks := make([]Task, 0, len(items))
    for _, item := range items {
        tasks = append(tasks, Task{
            Title:       item.Title,
            Description: item.Description,
            Slug:        ToSlug(item.Title),
        })
    }
    return tasks, nil
}
```

#### Example: REST API Source

```go
package source

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// APISource fetches tasks from a REST API endpoint.
type APISource struct {
    Endpoint string            // Full URL to GET tasks from
    Headers  map[string]string // Auth headers, content-type, etc.
}

func (s *APISource) Name() string { return "api" }

func (s *APISource) FetchTasks() ([]Task, error) {
    req, err := http.NewRequest("GET", s.Endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("api request build: %w", err)
    }
    for k, v := range s.Headers {
        req.Header.Set(k, v)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("api request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("api returned %d: %s", resp.StatusCode, string(body))
    }

    var items []struct {
        Title       string `json:"title"`
        Description string `json:"description"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
        return nil, fmt.Errorf("api response parse: %w", err)
    }

    tasks := make([]Task, 0, len(items))
    for _, item := range items {
        tasks = append(tasks, Task{
            Title:       item.Title,
            Description: item.Description,
            Slug:        ToSlug(item.Title),
        })
    }
    return tasks, nil
}
```

### 3. Add config fields

Add any new configuration your source needs to `internal/config/config.go`:

```go
type Config struct {
    // ... existing fields ...

    // MCP Source
    MCPServer string // MCP server name
    MCPTool   string // MCP tool to invoke

    // API Source
    APIEndpoint string // REST endpoint URL
    APIToken    string // Bearer token
}
```

### 4. Add CLI flags

Register the flags in `cmd/root.go` inside `func init()`:

```go
// MCP Source
rootCmd.Flags().StringVar(&cfg.MCPServer, "mcp-server", "",
    "MCP server name to fetch tasks from")
rootCmd.Flags().StringVar(&cfg.MCPTool, "mcp-tool", "get_tasks",
    "MCP tool name to call for fetching tasks")

// API Source
rootCmd.Flags().StringVar(&cfg.APIEndpoint, "api-endpoint", "",
    "REST API endpoint to fetch tasks from")
rootCmd.Flags().StringVar(&cfg.APIToken, "api-token", "",
    "Bearer token for the API source")
```

### 5. Wire up the routing

Add your source to `ResolveSource` in `internal/orchestrator/orchestrator.go`. The function returns the first matching source based on which flags the user provided:

```go
func ResolveSource(cfg config.Config) (source.Source, error) {
    // MCP source
    if cfg.MCPServer != "" {
        return &source.MCPSource{
            ServerName: cfg.MCPServer,
            ToolName:   cfg.MCPTool,
        }, nil
    }

    // API source
    if cfg.APIEndpoint != "" {
        return &source.APISource{
            Endpoint: cfg.APIEndpoint,
            Headers:  map[string]string{"Authorization": "Bearer " + cfg.APIToken},
        }, nil
    }

    // File source (default fallback)
    inputFile := cfg.InputFile
    if inputFile == "" {
        return nil, fmt.Errorf("no task source specified — use --input <path>")
    }
    // ... existing auto-detect logic ...
    return &source.FileSource{Path: inputFile}, nil
}
```

### 6. Add tests

Create `internal/source/<name>_test.go`. At minimum, test that `Name()` returns the expected value and `FetchTasks()` handles both success and error cases:

```go
package source

import "testing"

func TestMCPSource_Name(t *testing.T) {
    s := &MCPSource{ServerName: "test"}
    if s.Name() != "mcp" {
        t.Errorf("Name() = %q; want %q", s.Name(), "mcp")
    }
}
```

### 7. Update the input check in cmd/root.go

In the root command's `RunE`, update the "no input" guard to account for your new source flag:

```go
hasInput := cmd.Flags().Changed("prd") || cmd.Flags().Changed("input") ||
    cmd.Flags().Changed("plan") || cmd.Flags().Changed("mcp-server") ||
    cmd.Flags().Changed("api-endpoint")
if !hasInput {
    // show info panel and exit
}
```

---

## Architecture Summary

```
┌─────────────────────────────────────────────────┐
│  cmd/root.go                                    │
│  CLI flags → config.Config                      │
└─────────────┬───────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────┐
│  orchestrator.ResolveSource(cfg)                │
│  Routes config flags to the correct Source impl │
└─────────────┬───────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────┐
│  internal/source/                               │
│  ┌──────────────┐  ┌──────────────┐             │
│  │ file.go      │  │ mcp.go       │  ...        │
│  │ FileSource   │  │ MCPSource    │             │
│  └──────────────┘  └──────────────┘             │
│  All implement: Source.FetchTasks() → []Task    │
└─────────────┬───────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────┐
│  orchestrator.Run()                             │
│  worktrees → agent.Invoke() → PRs → summary    │
└─────────────────────────────────────────────────┘
```

## Key Rules

1. **Sources only produce tasks.** A source's job is to return `[]Task`. It should not create worktrees, invoke agents, or push branches — the orchestrator handles all of that.
2. **Use `ToSlug()` for branch safety.** Always convert titles to slugs via `source.ToSlug(title)` so branch names are valid git refs.
3. **One task = one worktree = one agent.** Each `Task` gets its own isolated git worktree and agent invocation. If your source naturally produces multiple tasks (e.g. a project board), return them all from `FetchTasks()`.
4. **Per-task model overrides are optional.** If `Task.Model` is empty, the orchestrator uses the global `--model` default. Set it if your source knows certain tasks need a specific model.
5. **Keep the file source as the default fallback.** New sources should be checked before the file source in `ResolveSource`, with the file source remaining as the catch-all.
