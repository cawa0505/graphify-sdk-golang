# API Reference — graphify-sdk-go

## Installation

```bash
go get github.com/cawa0505/graphify-sdk-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/cawa0505/graphify-sdk-go"
)

func main() {
    client := graphify.NewClient("/path/to/your/project")
    defer client.Stop()

    summary, err := client.GraphSummary()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Nodes: %d, Edges: %d, Languages: %v\n",
        summary.TotalNodes, summary.TotalEdges, summary.Languages)
}
```

## Client Configuration

### NewClient

```go
func NewClient(projectPath string, opts ...ClientOption) *Client
```

Creates a new Graphify client. `projectPath` is the root of the project to analyze.
If empty, defaults to the current working directory. The client automatically
derives a workspace key from the project path using crc32.

### Options

#### WithBinaryPath

```go
func WithBinaryPath(path string) ClientOption
```

Sets the path to the `graphify` binary. If not set, the SDK looks for
`graphify` on the system PATH.

```go
client := graphify.NewClient(".", graphify.WithBinaryPath("/usr/local/bin/graphify"))
```

#### WithTimeout

```go
func WithTimeout(timeout time.Duration) ClientOption
```

Sets the I/O timeout for MCP requests. Default: 30 seconds.

```go
client := graphify.NewClient(".", graphify.WithTimeout(60*time.Second))
```

### Lifecycle

| Method | Description |
|--------|-------------|
| `Start() error` | Explicitly starts the transport (auto-starts on first request) |
| `Stop()` | Stops the transport and cleans up the subprocess |
| `IsRunning() bool` | Reports whether the transport subprocess is running |

```go
client := graphify.NewClient("/path/to/project")
defer client.Stop() // Always clean up
```

## Core Graph Tools

### GraphSummary

```go
func (c *Client) GraphSummary() (*types.GraphSummary, error)
```

Returns high-level topology metrics of the knowledge graph.

**Returns:**
- `TotalNodes` — total number of nodes in the graph
- `TotalEdges` — total number of edges in the graph
- `Languages` — list of programming languages detected

```go
summary, err := client.GraphSummary()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d nodes, %d edges\n", summary.TotalNodes, summary.TotalEdges)
```

### QueryGraph

```go
func (c *Client) QueryGraph(question string) (*types.GraphOutput, error)
```

Performs a BFS traversal of the knowledge graph by natural language question.

**Parameters:**
- `question` — natural language query describing what to find

**Returns:**
- `GraphOutput` containing `Nodes`, `Edges`, and `Metadata`

```go
result, err := client.QueryGraph("find all authentication-related code")
if err != nil {
    log.Fatal(err)
}
for _, node := range result.Nodes {
    fmt.Printf("  %s (%s) in %s\n", node.Label, node.Kind, node.SourceFile)
}
```

### QueryNode

```go
func (c *Client) QueryNode(nodeID string, depth ...int) (*types.GraphOutput, error)
```

Queries a node by ID with optional traversal depth.

**Parameters:**
- `nodeID` — canonical node ID (e.g., `"src/main.go:function:main"`)
- `depth` — optional traversal depth (default: 1)

**Returns:**
- `GraphOutput` containing the node and its neighbors up to the specified depth

```go
// Query with default depth
result, err := client.QueryNode("src/auth.go:function:Login")

// Query with depth 3
result, err := client.QueryNode("src/auth.go:struct:AuthService", 3)
```

### TracePath

```go
func (c *Client) TracePath(from, to string) ([]string, error)
```

Finds the shortest path between two nodes in the knowledge graph.

**Parameters:**
- `from` — starting node ID
- `to` — target node ID

**Returns:**
- `[]string` — ordered list of node IDs forming the path

```go
path, err := client.TracePath(
    "src/models/user.go:struct:User",
    "src/http/handlers/auth.go:function:Login",
)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Dependency path:", strings.Join(path, " → "))
```

### ReindexFile

```go
func (c *Client) ReindexFile(filePath string) (*types.ReindexResult, error)
```

Reindexes a file into the knowledge graph, updating its nodes and edges.

**Parameters:**
- `filePath` — path to the file to reindex

**Returns:**
- `ReindexResult` with `Status`, `TotalNodes`, `TotalEdges`

```go
result, err := client.ReindexFile("src/new_feature.go")
if err != nil {
    log.Fatal(err)
}
if result.IsSuccess() {
    fmt.Printf("Reindexed: %d nodes, %d edges\n", result.TotalNodes, result.TotalEdges)
}
```

## Memory Query

### MemoryQuery

```go
func (c *Client) MemoryQuery(query string, limit ...int) (*types.MemoryQueryResult, error)
```

Performs a semantic memory search against the knowledge graph.

**Parameters:**
- `query` — natural language search query
- `limit` — optional maximum number of results (default: all)

**Returns:**
- `MemoryQueryResult` with `Status` and `Nodes`

```go
result, err := client.MemoryQuery("find user authentication", 10)
if err != nil {
    log.Fatal(err)
}
if result.IsFound() {
    for _, node := range result.Nodes {
        fmt.Printf("%s (%s) in %s\n", node.Label, node.Kind, node.SourceFile)
    }
} else {
    fmt.Println("No results found")
}
```

## Relay / Handoff Tools

### RelayInit

```go
func (c *Client) RelayInit(projectContext string, kind ...string) (map[string]any, error)
```

Initializes a relay session for cross-session state handoff.

**Parameters:**
- `projectContext` — project context description
- `kind` — optional template kind (backend, frontend, infra)

```go
result, err := client.RelayInit("Backend API service with auth module", "backend")
```

### RelaySave

```go
func (c *Client) RelaySave(params map[string]any) (map[string]any, error)
```

Saves the current session state to relay.

```go
result, err := client.RelaySave(map[string]any{
    "volatile": "Implemented auth middleware",
    "phase":    "EXECUTING",
    "conf":     4,
})
```

### RelayClose

```go
func (c *Client) RelayClose(repo, next string) (map[string]any, error)
```

Closes a relay session with a handoff document.

**Parameters:**
- `repo` — repository name
- `next` — next session starter text

```go
result, err := client.RelayClose("my-project", "Continue with user registration")
```

### RelaySwitch

```go
func (c *Client) RelaySwitch(repo string, kind ...string) (map[string]any, error)
```

Switches the active relay baton to another repository.

```go
result, err := client.RelaySwitch("other-project", "backend")
```

### RelayResume

```go
func (c *Client) RelayResume(repo string, kind ...string) (map[string]any, error)
```

Resumes a relay session for a repository.

```go
result, err := client.RelayResume("my-project")
```

### RelayStatus

```go
func (c *Client) RelayStatus() (*types.RelayStatus, error)
```

Returns the relay status summary with active repos, baton, and spec drift.

```go
status, err := client.RelayStatus()
if err != nil {
    log.Fatal(err)
}
```

### RelayAdd

```go
func (c *Client) RelayAdd(file, repo string) (map[string]any, error)
```

Ingests a TODO/handoff document into the relay system.

**Parameters:**
- `file` — path to the document file
- `repo` — repository name to associate

```go
result, err := client.RelayAdd("HANDOFF.md", "my-project")
```

## OpenDoc Tools

### OpenDocIndex

```go
func (c *Client) OpenDocIndex(docPaths ...string) (map[string]any, error)
```

Indexes all spec blocks in the workspace. Accepts optional explicit doc paths.

```go
// Index all .md files in workspace
result, err := client.OpenDocIndex()

// Index specific files
result, err := client.OpenDocIndex("docs/api.md", "docs/design.md")
```

### OpenDocGetContext

```go
func (c *Client) OpenDocGetContext(symbol string) (map[string]any, error)
```

Returns spec blocks documenting a code symbol.

```go
result, err := client.OpenDocGetContext("crate::auth::verify_token")
```

### OpenDocAuditDrift

```go
func (c *Client) OpenDocAuditDrift() (map[string]any, error)
```

Audits doc-side drift — checks if documented specs match current code.

```go
result, err := client.OpenDocAuditDrift()
```

## Review Tools

### ReviewIngest

```go
func (c *Client) ReviewIngest(payload string) (map[string]any, error)
```

Imports a CRG review payload into the review registry.

**Parameters:**
- `payload` — path to the IngestPayload JSON file

```go
result, err := client.ReviewIngest("review-payload.json")
```

### ReviewGetContext

```go
func (c *Client) ReviewGetContext(node string) (map[string]any, error)
```

Queries unresolved reviews for a code symbol.

```go
result, err := client.ReviewGetContext("src/auth.rs:function:verify_token")
```

### ReviewResolve

```go
func (c *Client) ReviewResolve(reviewID, reason string) (map[string]any, error)
```

Marks a review as resolved.

**Parameters:**
- `reviewID` — the review's unique identifier
- `reason` — resolution description

```go
result, err := client.ReviewResolve("rev_abc123", "Fixed in commit abc123")
```

### ReviewSearchCrg

```go
func (c *Client) ReviewSearchCrg(base ...string) (map[string]any, error)
```

Searches CRG for changed functions and binds them as review points.

**Parameters:**
- `base` — optional git diff base ref (default: HEAD~1)

```go
result, err := client.ReviewSearchCrg("HEAD~3")
```

## Telemetry Tools

### TelemetryIngest

```go
func (c *Client) TelemetryIngest(source string, pathOrDracoParams ...string) (map[string]any, error)
```

Imports telemetry metrics into the telemetry registry.

**Parameters:**
- `source` — `"file"` (local JSON file) or `"draco-mcp"` (live poll)
- `pathOrDracoParams` — JSON file path (when source="file")

```go
result, err := client.TelemetryIngest("file", "telemetry.json")
result, err := client.TelemetryIngest("draco-mcp")
```

### TelemetryGetContext

```go
func (c *Client) TelemetryGetContext(node string, includeImpactRadius ...bool) (map[string]any, error)
```

Queries telemetry bindings for a node.

**Parameters:**
- `node` — canonical node ID
- `includeImpactRadius` — optionally expand upstream callers impact radius

```go
result, err := client.TelemetryGetContext("src/db/query.rs:function:query_users", true)
```

## Coverage Tools

### CoverageIngest

```go
func (c *Client) CoverageIngest(format, data string) (map[string]any, error)
```

Imports coverage data into the coverage registry.

**Parameters:**
- `format` — `"lcov"` or `"json"` (cobertura)
- `data` — raw coverage data string

```go
result, err := client.CoverageIngest("lcov", lcovData)
```

### CoverageGetContext

```go
func (c *Client) CoverageGetContext(node string) (*types.CoverageResult, error)
```

Queries coverage information for a node.

**Returns:**
- `CoverageResult` with `Node`, `CoveredLines`, `TotalLines`, `LineRate`

```go
coverage, err := client.CoverageGetContext("src/auth.rs:function:verify_token")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Coverage: %.1f%% (%d/%d lines)\n",
    coverage.Percentage(), coverage.CoveredLines, coverage.TotalLines)
```

### CoverageBlindspots

```go
func (c *Client) CoverageBlindspots() (map[string]any, error)
```

Lists all nodes with coverage below 50%.

```go
result, err := client.CoverageBlindspots()
```

## Plugin Gateway

### PluginNotify

```go
func (c *Client) PluginNotify(kind string) (map[string]any, error)
```

Broadcasts a graph update notification to all healthy plugin subprocesses.

**Parameters:**
- `kind` — notification kind (`"indexed"`, `"extracted"`, `"manual"`)

```go
result, err := client.PluginNotify("indexed")
```

## Workspace Context

```go
func (c *Client) WorkspaceContext() types.WorkspaceContext
func (c *Client) ProjectPath() string
```

Returns workspace context information.

```go
ctx := client.WorkspaceContext()
fmt.Printf("Workspace: %s (key: %s)\n", ctx.WorkspaceName, ctx.WorkspaceKey)
fmt.Printf("Project path: %s\n", client.ProjectPath())
```

## Types Reference

### GraphSummary

```go
type GraphSummary struct {
    TotalNodes int      `json:"total_nodes"`
    TotalEdges int      `json:"total_edges"`
    Languages  []string `json:"languages"`
}
```

### GraphOutput

```go
type GraphOutput struct {
    Nodes    []Node        `json:"nodes"`
    Edges    []Edge        `json:"edges"`
    Metadata GraphMetadata `json:"metadata"`
}
```

### GraphMetadata

```go
type GraphMetadata struct {
    Version      string         `json:"version"`
    GeneratedAt  string         `json:"generated_at"`
    TotalNodes   int            `json:"total_nodes"`
    TotalEdges   int            `json:"total_edges"`
    Languages    []string       `json:"languages"`
    InputTokens  int            `json:"input_tokens"`
    OutputTokens int            `json:"output_tokens"`
    PluginData   map[string]any `json:"plugin_data,omitempty"`
}
```

### Node

```go
type Node struct {
    ID          string         `json:"id"`
    Label       string         `json:"label"`
    FileType    string         `json:"file_type"`
    Kind        string         `json:"kind"`
    Language    string         `json:"language"`
    SourceFile  string         `json:"source_file"`
    StartLine   int            `json:"start_line"`
    EndLine     int            `json:"end_line"`
    DocComment  *string        `json:"doc_comment,omitempty"`
    Description *string        `json:"description,omitempty"`
    Metadata    map[string]any `json:"metadata,omitempty"`
}
```

### Edge

```go
type Edge struct {
    Source         string  `json:"source"`
    Target         string  `json:"target"`
    Relation       string  `json:"relation"`
    SourceFile     string  `json:"source_file"`
    Confidence     string  `json:"confidence"`
    SourceLocation string  `json:"source_location"`
    Description    *string `json:"description,omitempty"`
}
```

### CoverageResult

```go
type CoverageResult struct {
    Node         string  `json:"node"`
    CoveredLines int     `json:"covered_lines"`
    TotalLines   int     `json:"total_lines"`
    LineRate     float64 `json:"line_rate"`
}

func (c *CoverageResult) Percentage() float64  // Returns LineRate * 100
```

### ReindexResult

```go
type ReindexResult struct {
    Status     string `json:"status"`
    TotalNodes int    `json:"total_nodes"`
    TotalEdges int    `json:"total_edges"`
}

func (r *ReindexResult) IsSuccess() bool
```

### MemoryQueryResult

```go
type MemoryQueryResult struct {
    Status string `json:"status"`
    Nodes  []Node `json:"nodes,omitempty"`
}

func (m *MemoryQueryResult) IsFound() bool
```

### RelayStatus

```go
type RelayStatus struct {
    Repos       map[string]any `json:"repos"`
    ActiveBaton *string        `json:"active_baton,omitempty"`
    SpecDrift   *string        `json:"spec_drift,omitempty"`
    LastUpdate  *string        `json:"last_update,omitempty"`
}
```

### WorkspaceContext

```go
type WorkspaceContext struct {
    WorkspaceKey  string `json:"workspace_key"`
    WorkspaceName string `json:"workspace_name"`
    RootPath      string `json:"root_path"`
    Timestamp     int64  `json:"timestamp"`
}
```

## Error Types

```go
type GraphifyError struct{ Msg string }

type TransportError struct{ GraphifyError }   // I/O and process errors
type ProtocolError struct{ GraphifyError }    // JSON-RPC protocol errors

type EngineError struct {
    GraphifyError
    Code      int
    ErrorData map[string]any
}
```

Use `errors.As` for type checking:

```go
_, err := client.GraphSummary()
if err != nil {
    var engineErr *graphify.EngineError
    if errors.As(err, &engineErr) {
        fmt.Printf("Engine error (code %d): %s\n", engineErr.Code, engineErr.Msg)
    }
}
```