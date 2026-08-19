# Design: graphify-sdk-go v1

## Architecture Overview

```
┌─────────────────────────────────┐
│   Go Application / CLI          │
│   (net/http, gin, cobra, etc.)  │
├─────────────────────────────────┤
│   graphify-sdk-go               │
│                                 │
│  ┌───────────────────────────┐  │
│  │  GraphifyClient (Client)  │  │  ← Query Graphify
│  │  - GraphSummary()         │  │
│  │  - QueryGraph()           │  │
│  │  - TracePath()            │  │
│  │  - MemoryQuery()          │  │
│  │  - 24 public methods      │  │
│  └──────┬────────────────────┘  │
│         │                       │
│  ┌──────▼────────────────────┐  │
│  │  Transport                 │  │  ← Stdio/JSON-RPC
│  │  - Start()                 │  │
│  │  - Call()                  │  │
│  │  - Stop()                  │  │
│  └──────┬────────────────────┘  │
│         │                       │
│  ┌──────▼────────────────────┐  │
│  │  types (DTOs)              │  │  ← Node, Edge, GraphOutput, etc.
│  └───────────────────────────┘  │
│                                 │
│  ┌───────────────────────────┐  │
│  │  plugin.Host (Plugin SDK) │  │  ← Graphify plugin runner
│  │  - RegisterTool()         │  │
│  │  - Run()                  │  │
│  │  - handleInitialize()     │  │
│  │  - handleToolsCall()      │  │
│  └───────────────────────────┘  │
│                                 │
│  ┌───────────────────────────┐  │
│  │  Errors                    │  │  ← Typed error hierarchy
│  └───────────────────────────┘  │
└──────────┬──────────────────────┘
           │ Stdio (stdin/stdout)
┌──────────▼──────────────────────┐
│  graphify (Rust binary)      │
└─────────────────────────────────┘
```

## Package Structure

```
graphify-sdk-golang/
├── client.go          # Public API — wraps all 24+ MCP tools
├── transport.go       # Stdio/JSON-RPC transport layer
├── errors.go          # Typed error hierarchy
├── plugin/
│   └── host.go        # Plugin SDK — JSON-RPC stdio host
├── types/
│   └── types.go       # Data transfer objects (15+ types)
├── docs/
│   ├── design.md      # Architecture design document
│   ├── api-reference.md  # Full API reference
│   └── plugin-sdk.md  # Plugin SDK guide
├── AGENTS.md          # Agent instructions
├── go.mod             # Zero external dependencies (stdlib only)
├── .gitignore
├── README.md
└── README.zh-TW.md
```

## Transport Layer

### Process Lifecycle

```go
transport := NewTransport("graphify", 30*time.Second, "/path/to/project")
transport.Start()    // Spawns process, opens pipes
// ... use ...
transport.Stop()     // Closes pipes, terminates process
```

### JSON-RPC Communication

The Transport implements MCP JSON-RPC 2.0 over stdin/stdout:

```json
→ {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graphify_graph_summary","arguments":{}}}
← {"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"total_nodes\":42,\"total_edges\":156}"}]}}
```

### Request/Response Flow

1. Build JSON-RPC 2.0 request with auto-incrementing ID
2. Write to subprocess stdin (newline-delimited JSON)
3. Read JSON-RPC response from subprocess stdout (bufio.Scanner)
4. Parse response, check for protocol errors
5. Extract text content from MCP content items, parse inner JSON
6. Return decoded result map

### Thread Safety

Transport uses `sync.Mutex` to protect shared state:
- `mu.Lock()` around `requestID` increment
- `mu.Lock()` around process lifecycle (start/stop)
- Read operations use `bufio.Scanner` with mutex guard

### Error Handling

- **Process spawn failure**: `TransportError` with wrapped error
- **I/O timeout**: configurable (default 30s), returns `TransportError`
- **stdout closed prematurely**: `TransportError` with "process exited" message
- **JSON-RPC error response**: `EngineError` with code and message
- **Malformed response**: `ProtocolError`

## Client Layer (GraphifyClient)

### Workspace Key Auto-Detection

```go
client := NewClient("/path/to/project")
```

Derivation (mirrors Rust): `crc32.ChecksumIEEE` of canonicalized root path → hex string.
Go's standard library `hash/crc32` provides IEEE polynomial, matching the PHP SDK's
`crc32()` function for cross-language consistency.

### Lazy Initialization

The subprocess is spawned on the first tool call, not in the constructor.
This allows configuration before connection and avoids unnecessary process
spawning for configuration-only code paths.

### Tool Method Mapping

Every `graphify` tool maps to a public method (24 total):

| Tool Name | Go Method | Returns |
|-----------|-----------|---------|
| `graphify_graph_summary` | `GraphSummary()` | `*types.GraphSummary` |
| `graphify_graph_query` | `QueryGraph(question)` | `*types.GraphOutput` |
| `graphify_graph_query_node` | `QueryNode(nodeID, ...depth)` | `*types.GraphOutput` |
| `graphify_graph_trace_path` | `TracePath(from, to)` | `[]string` |
| `graphify_graph_reindex` | `ReindexFile(filePath)` | `*types.ReindexResult` |
| `graphify_memory_query` | `MemoryQuery(query, ...limit)` | `*types.MemoryQueryResult` |
| `graphify_relay_init` | `RelayInit(projectContext, ...kind)` | `map[string]any` |
| `graphify_relay_save` | `RelaySave(params)` | `map[string]any` |
| `graphify_relay_close` | `RelayClose(repo, next)` | `map[string]any` |
| `graphify_relay_switch` | `RelaySwitch(repo, ...kind)` | `map[string]any` |
| `graphify_relay_resume` | `RelayResume(repo, ...kind)` | `map[string]any` |
| `graphify_relay_status` | `RelayStatus()` | `*types.RelayStatus` |
| `graphify_relay_add` | `RelayAdd(file, repo)` | `map[string]any` |
| `graphify_opendoc_index` | `OpenDocIndex(...docPaths)` | `map[string]any` |
| `graphify_opendoc_get_context` | `OpenDocGetContext(symbol)` | `map[string]any` |
| `graphify_opendoc_audit_drift` | `OpenDocAuditDrift()` | `map[string]any` |
| `graphify_review_ingest` | `ReviewIngest(payload)` | `map[string]any` |
| `graphify_review_get_context` | `ReviewGetContext(node)` | `map[string]any` |
| `graphify_review_resolve` | `ReviewResolve(reviewID, reason)` | `map[string]any` |
| `graphify_review_search_crg` | `ReviewSearchCrg(...base)` | `map[string]any` |
| `graphify_telemetry_ingest` | `TelemetryIngest(source, ...path)` | `map[string]any` |
| `graphify_telemetry_get_context` | `TelemetryGetContext(node, ...radius)` | `map[string]any` |
| `graphify_coverage_ingest` | `CoverageIngest(format, data)` | `map[string]any` |
| `graphify_coverage_get_context` | `CoverageGetContext(node)` | `*types.CoverageResult` |
| `graphify_coverage_blindspots` | `CoverageBlindspots()` | `map[string]any` |
| `graphify_plugin_notify` | `PluginNotify(kind)` | `map[string]any` |

## Plugin SDK Layer (plugin.Host)

### Role

`plugin.Host` is the **inbound** side of the SDK — it lets Go programs run as
Graphify plugins via IPC subprocess. Graphify Core spawns the Go binary,
communicates via JSON-RPC over stdin/stdout, and the Host dispatches tool calls
to registered Go handlers.

### Protocol

Implements MCP JSON-RPC subset:

| Method | Purpose |
|--------|---------|
| `initialize` | Returns protocol version, capabilities, and tool list |
| `tools/list` | Returns registered tools with schemas |
| `tools/call` | Dispatches to registered handler, returns result |
| `notifications/*` | Silently accepted (no response expected) |

### Plugin Host Lifecycle

```
Graphify Core (Rust)               Go Plugin Process
       │                                  │
       │  exec.Command("my-plugin")        │
       │─────────────────────────────────>│
       │                                  │
       │  {"method":"initialize",...}     │
       │─────────────────────────────────>│
       │  {"tools":["my_tool",...]}        │
       │<─────────────────────────────────│
       │                                  │
       │  {"method":"tools/call",...}     │
       │─────────────────────────────────>│
       │  {"result":{...}}                │
       │<─────────────────────────────────│
       │                                  │
       │  close stdin (EOF)               │
       │─────────────────────────────────>│
       │  exit(0)                         │
       │<─────────────────────────────────│
```

### Usage

```go
package main

import "github.com/cawa0505/graphify-sdk-go/plugin"

func main() {
    host := plugin.NewHost()
    host.RegisterTool("analyze", plugin.ToolSchema{
        Description: "Analyze project structure",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "path": map[string]any{
                    "type":        "string",
                    "description": "Project path to analyze",
                },
            },
            "required": []string{"path"},
        },
    }, func(args map[string]any) (map[string]any, error) {
        // Your Go logic here — no Rust, no WASM
        return analyzeProject(args["path"].(string))
    })
    host.Run() // Blocks, reads stdin, writes stdout
}
```

## DTO Design

### Types Package

All data transfer objects live in the `types` package:

```go
type Node struct {
    ID          string
    Label       string
    FileType    string
    Kind        string
    Language    string
    SourceFile  string
    StartLine   int
    EndLine     int
    DocComment  *string
    Description *string
    Metadata    map[string]any
}

type Edge struct {
    Source         string
    Target         string
    Relation       string
    SourceFile     string
    Confidence     string
    SourceLocation string
    Description    *string
}

type GraphOutput struct {
    Nodes    []Node
    Edges    []Edge
    Metadata GraphMetadata
}

type GraphSummary struct {
    TotalNodes int
    TotalEdges int
    Languages  []string
}
```

### JSON Round-Trip Conversion

The SDK uses Go's `encoding/json` for type-safe serialization:

```go
// mapTo converts raw map to typed struct via JSON marshal/unmarshal
func mapTo[T any](data map[string]any) (*T, error) {
    b, _ := json.Marshal(data)
    var out T
    json.Unmarshal(b, &out)
    return &out, nil
}
```

This is the idiomatic Go approach for dynamic-to-typed conversion with zero
external dependencies.

## Error Handling

### Error Hierarchy

```
GraphifyError (base)
├── TransportError    — Process spawn/pipe I/O errors
├── ProtocolError     — JSON-RPC parse/error response
└── EngineError       — graphify returned error
    ├── Code          — JSON-RPC error code
    └── ErrorData     — Additional error context
```

## Key Design Decisions

1. **Zero external dependencies**: No MCP client library, no third-party packages.
   Go's standard library `os/exec`, `encoding/json`, `bufio`, `hash/crc32` are
   sufficient for Stdio/JSON-RPC. This keeps the SDK installable without
   dependency conflicts.

2. **Synchronous API**: Go's goroutines make async tempting, but the underlying
   MCP protocol is request-response. The SDK exposes a synchronous interface,
   which is simpler and sufficient for most use cases. Users can wrap in goroutines
   if needed.

3. **Generics for type-safe conversion**: Go 1.22+ generics power the `mapTo[T]`
   helper, providing compile-time type safety for DTO conversion without
   runtime reflection overhead.

4. **Lazy process start**: The transport subprocess is spawned on first request,
   not in constructor, to allow configuration before connection.

5. **crc32 workspace key**: Cross-platform stable hash from `hash/crc32` (IEEE
   polynomial), matching the PHP SDK's `crc32()` for consistency across the
   SDK family.

6. **Mutex-guarded transport**: Thread-safe for concurrent access patterns,
   though the SDK is designed for single-threaded use.

7. **Variadic options**: Go idiomatic functional options pattern for optional
   parameters (WithBinaryPath, WithTimeout).

## SDK Family Alignment

All Graphify SDKs implement the same tool method set (based on `graphify`
tools), ensuring cross-language API consistency. The `graphify` package name
and method signatures follow the same conventions as the PHP SDK:

| SDK | Package | Client Init | Method Pattern |
|-----|---------|-------------|----------------|
| PHP | `Graphify\Sdk` | `new GraphifyClient(...)` | `camelCase()` |
| Go  | `graphify` | `NewClient(...)` | `PascalCase()` |

## Future Considerations

### Async/Await Support

The transport layer is designed to support future async/context-based variants:
- `CallContext(ctx context.Context, ...)` — already has the foundation
- Timeout via `context.WithTimeout` — aware of context deadlines
- Connection pooling for multiple simultaneous requests

### WASM Plugin Support

The plugin SDK currently supports IPC mode only. WASM support can be added
later through the same `PluginInterface` pattern, with a WASM runtime
(like wazero) replacing the subprocess lifecycle.

### Concurrency Model

Currently single-threaded. A future version could add:
- Mutex-locked request/response for safe concurrent access
- Request pipelining for batch operations
- Graceful shutdown via context cancellation