# Graphify Go SDK

Official Go SDK for [Graphify](https://github.com/cawa0505/GraphifySDK) — access
knowledge graph capabilities over the MCP (Model Context Protocol) via Stdio/JSON-RPC.

## Documentation

- **[docs/design.md](docs/design.md)** — Architecture overview, transport layer, DTO design, key decisions
- **[docs/api-reference.md](docs/api-reference.md)** — Complete API reference with examples and type reference
- **[docs/plugin-sdk.md](docs/plugin-sdk.md)** — Plugin SDK guide for building Graphify plugins in Go

## Requirements

- Go 1.22+
- `graphify-mcp` binary on PATH (or configured path)

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

	// Graph summary
	summary, err := client.GraphSummary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Nodes: %d, Edges: %d\n", summary.TotalNodes, summary.TotalEdges)

	// Semantic memory query
	result, err := client.MemoryQuery("find user authentication")
	if err != nil {
		log.Fatal(err)
	}
	if result.IsFound() {
		for _, node := range result.Nodes {
			fmt.Printf("%s (%s) in %s\n", node.Label, node.Kind, node.SourceFile)
		}
	}

	// Trace dependency path
	path, err := client.TracePath(
		"src/models/user.go:struct:User",
		"src/http/handlers/auth.go:function:Login",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Path:", path)

	// Query node with depth
	graph, err := client.QueryNode(
		"src/services/auth.go:struct:AuthService",
		2,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d related nodes\n", len(graph.Nodes))
}
```

## API Reference

The SDK wraps all 24+ `graphify-mcp` tools.

### Core Graph

| Method | Description | Returns |
|--------|-------------|---------|
| `GraphSummary()` | Topology metrics | `*types.GraphSummary` |
| `QueryGraph(question)` | BFS traversal | `*types.GraphOutput` |
| `QueryNode(nodeId, ...depth)` | Node query | `*types.GraphOutput` |
| `TracePath(from, to)` | Shortest path | `[]string` |
| `ReindexFile(filePath)` | Reindex file | `*types.ReindexResult` |

### Memory & Relay

| Method | Description | Returns |
|--------|-------------|---------|
| `MemoryQuery(query, ...limit)` | Semantic search | `*types.MemoryQueryResult` |
| `RelayInit(projectContext, ...kind)` | Init handoff | `map[string]any` |
| `RelaySave(params)` | Save state | `map[string]any` |
| `RelayClose(repo, next)` | Close handoff | `map[string]any` |
| `RelaySwitch(repo, ...kind)` | Switch repo | `map[string]any` |
| `RelayResume(repo, ...kind)` | Resume | `map[string]any` |
| `RelayStatus()` | Status summary | `*types.RelayStatus` |
| `RelayAdd(file, repo)` | Ingest doc | `map[string]any` |

### OpenDoc

| Method | Description | Returns |
|--------|-------------|---------|
| `OpenDocIndex(...docPaths)` | Index spec blocks | `map[string]any` |
| `OpenDocGetContext(symbol)` | Get symbol docs | `map[string]any` |
| `OpenDocAuditDrift()` | Audit drift | `map[string]any` |

### Review

| Method | Description | Returns |
|--------|-------------|---------|
| `ReviewIngest(payload)` | Import review | `map[string]any` |
| `ReviewGetContext(node)` | Query reviews | `map[string]any` |
| `ReviewResolve(reviewID, reason)` | Resolve review | `map[string]any` |
| `ReviewSearchCrg(...base)` | Search CRG | `map[string]any` |

### Telemetry & Coverage

| Method | Description | Returns |
|--------|-------------|---------|
| `TelemetryIngest(source, ...path)` | Import metrics | `map[string]any` |
| `TelemetryGetContext(node, ...radius)` | Query telemetry | `map[string]any` |
| `CoverageIngest(format, data)` | Import coverage | `map[string]any` |
| `CoverageGetContext(node)` | Query coverage | `*types.CoverageResult` |
| `CoverageBlindspots()` | Low-coverage list | `map[string]any` |

### Plugin Gateway

| Method | Description | Returns |
|--------|-------------|---------|
| `PluginNotify(kind)` | Broadcast update | `map[string]any` |

## Architecture

```
Go App → GraphifyClient → Transport (Stdio/JSON-RPC) → graphify-mcp (Rust)
```

- **Zero external dependencies**: uses only the Go standard library
- **Synchronous API**: single-threaded, request-response over stdio
- **Auto workspace key**: derives from project path (mirrors Rust crc32 logic)
- **Lazy process start**: transport spawns `graphify-mcp` on first request

## Project Structure

```
graphify-sdk-golang/
├── client.go         # Public API — wraps all MCP tools
├── transport.go      # Stdio/JSON-RPC transport
├── errors.go         # Typed error hierarchy
├── plugin/
│   └── host.go       # Plugin SDK — JSON-RPC stdio host
├── types/
│   └── types.go      # Data transfer objects
├── go.mod
├── README.md
└── README.zh-TW.md
```

## Plugin SDK

The `plugin` package provides a JSON-RPC stdio host for building Graphify plugins
in Go. See [plugin/host.go](plugin/host.go) for details.

```go
host := plugin.NewHost()
host.RegisterTool("analyze", plugin.ToolSchema{
    Description: "Analyze project structure",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "path": map[string]any{
                "type":        "string",
                "description": "Project path",
            },
        },
    },
}, func(args map[string]any) (map[string]any, error) {
    return map[string]any{"status": "ok"}, nil
})
host.Run()
```

## License

MIT