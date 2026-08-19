# Plugin SDK Guide — graphify-sdk-go

## Overview

The `plugin` package provides the **inbound** side of the Graphify SDK — it lets
Go programs run as Graphify plugins via IPC subprocess. Graphify Core spawns
the Go binary, communicates via JSON-RPC over stdin/stdout, and the plugin host
dispatches tool calls to registered Go handlers.

This is the Go equivalent of the PHP SDK's `PluginHost` and mirrors the
Graphify Plugin Architecture described in the Graphify documentation.

## When to Use the Plugin SDK

Write a Graphify plugin in Go when you need to:

- Process graph data with custom logic (analysis, validation, transformation)
- Integrate with Go-specific libraries or systems
- Leverage Go's performance for compute-intensive operations
- Build a plugin that runs as an IPC subprocess (not WASM)

## Quick Start

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/cawa0505/graphify-sdk-go/plugin"
)

func main() {
    host := plugin.NewHost()

    host.RegisterTool("hello", plugin.ToolSchema{
        Description: "A simple hello world tool",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "name": map[string]any{
                    "type":        "string",
                    "description": "Name to greet",
                },
            },
        },
    }, func(args map[string]any) (map[string]any, error) {
        name := "World"
        if n, ok := args["name"].(string); ok {
            name = n
        }
        return map[string]any{
            "greeting": fmt.Sprintf("Hello, %s!", name),
        }, nil
    })

    host.Run()
}
```

Build and run:

```bash
go build -o my-plugin .
# Graphify Core will spawn this as a subprocess
```

## Protocol

The plugin host implements the MCP JSON-RPC subset:

| Method | Direction | Purpose |
|--------|-----------|---------|
| `initialize` | Core → Plugin | Handshake, returns capabilities and tool list |
| `tools/list` | Core → Plugin | Returns registered tools with schemas |
| `tools/call` | Core → Plugin | Dispatches tool call to registered handler |
| `notifications/*` | Core → Plugin | Silently accepted (no response) |

### Initialize

Graphify Core sends this on startup. The plugin responds with protocol version,
capabilities, and the list of registered tools.

Request:
```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
```

Response:
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "protocolVersion": "2024-11-05",
        "capabilities": {"tools": {}},
        "tools": [
            {
                "name": "hello",
                "description": "A simple hello world tool",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string", "description": "Name to greet"}
                    }
                }
            }
        ]
    }
}
```

### Tools/List

Request:
```json
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

Response:
```json
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "tools": [...]
    }
}
```

### Tools/Call

Request:
```json
{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
        "name": "hello",
        "arguments": {"name": "Graphify"}
    }
}
```

Response:
```json
{
    "jsonrpc": "2.0",
    "id": 3,
    "result": {
        "content": [
            {"type": "text", "text": "{\"greeting\":\"Hello, Graphify!\"}"}
        ]
    }
}
```

## API Reference

### NewHost

```go
func NewHost() *Host
```

Creates a new plugin host with no registered tools.

### RegisterTool

```go
func (h *Host) RegisterTool(name string, schema ToolSchema, handler ToolHandler) *Host
```

Registers a tool that this plugin provides. Chainable (returns `*Host`).

**Parameters:**
- `name` — unique tool name (used by Graphify Core to call this tool)
- `schema` — tool description and input schema
- `handler` — function that processes tool arguments and returns results

```go
host.RegisterTool("analyze", plugin.ToolSchema{
    Description: "Analyze project structure",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "path": map[string]any{
                "type":        "string",
                "description": "Path to analyze",
            },
        },
        "required": []string{"path"},
    },
}, func(args map[string]any) (map[string]any, error) {
    path := args["path"].(string)
    return analyzeProject(path)
})
```

### Run

```go
func (h *Host) Run()
```

Starts the JSON-RPC stdio listener (blocking). Reads newline-delimited JSON
from stdin, dispatches to registered handlers, writes responses to stdout.
Exits cleanly when stdin closes (parent process terminated the pipe).

### Stop

```go
func (h *Host) Stop()
```

Gracefully stops the listener loop.

## ToolSchema

```go
type ToolSchema struct {
    Description string
    InputSchema map[string]any
}
```

- `Description` — human-readable description of what the tool does
- `InputSchema` — JSON Schema describing the tool's arguments

## ToolHandler

```go
type ToolHandler func(args map[string]any) (map[string]any, error)
```

Function that receives tool arguments and returns a result map. Return an error
to signal failure; the host will convert it to a JSON-RPC error response.

## Error Handling

Plugin errors are automatically converted to JSON-RPC error responses:

```go
host.RegisterTool("fail", plugin.ToolSchema{
    Description: "Demonstrates error handling",
}, func(args map[string]any) (map[string]any, error) {
    return nil, fmt.Errorf("something went wrong")
})
```

The host sends:
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {"code": -32603, "message": "something went wrong"}
}
```

## Complete Example

### Plugin: File Analyzer

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/cawa0505/graphify-sdk-go/plugin"
)

func main() {
    host := plugin.NewHost()

    // Tool 1: Analyze file
    host.RegisterTool("analyze_file", plugin.ToolSchema{
        Description: "Analyze a file and return its metadata",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "path": map[string]any{
                    "type":        "string",
                    "description": "Path to the file",
                },
            },
            "required": []string{"path"},
        },
    }, func(args map[string]any) (map[string]any, error) {
        path := args["path"].(string)
        info, err := os.Stat(path)
        if err != nil {
            return nil, fmt.Errorf("cannot stat file: %w", err)
        }
        return map[string]any{
            "name":    filepath.Base(path),
            "size":    info.Size(),
            "is_dir":  info.IsDir(),
            "mode":    info.Mode().String(),
            "modtime": info.ModTime().Unix(),
            "ext":     filepath.Ext(path),
        }, nil
    })

    // Tool 2: Search files
    host.RegisterTool("search_files", plugin.ToolSchema{
        Description: "Search for files matching a pattern",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "root": map[string]any{
                    "type":        "string",
                    "description": "Root directory to search",
                },
                "pattern": map[string]any{
                    "type":        "string",
                    "description": "File pattern (e.g., *.go)",
                },
            },
            "required": []string{"root"},
        },
    }, func(args map[string]any) (map[string]any, error) {
        root := args["root"].(string)
        pattern := "*"
        if p, ok := args["pattern"].(string); ok {
            pattern = p
        }
        var matches []string
        filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return nil
            }
            if matched, _ := filepath.Match(pattern, info.Name()); matched {
                matches = append(matches, path)
            }
            return nil
        })
        return map[string]any{
            "matches": matches,
            "count":   len(matches),
        }, nil
    })

    host.Run()
}
```

## Debugging

Since the plugin communicates over stdin/stdout, use stderr for debug output:

```go
fmt.Fprintf(os.Stderr, "debug: processing %v\n", args)
```

All stderr output is captured by Graphify Core's plugin log.

## Best Practices

1. **Register all tools before calling Run()**: The host is single-threaded;
   register tools in init() or main() before entering the event loop.

2. **Use stderr for logging**: stdout is reserved for JSON-RPC communication.
   Write debug/info output to stderr.

3. **Handle all errors**: Return errors from handlers rather than panicking.
   The host converts panics to error responses, but explicit error handling
   is preferred.

4. **Validate input**: Check argument types and presence before using them.
   Return descriptive errors for invalid input.

5. **Keep handlers fast**: The plugin blocks while processing. For long-running
   operations, consider splitting into multiple tools or using background
   goroutines.

6. **Clean up resources**: Use defer to close files, connections, etc. The
   host sends a clean exit(0) when stdin closes.

## Comparison with PHP PluginHost

| Feature | Go (plugin.Host) | PHP (PluginHost) |
|---------|------------------|-------------------|
| Registration | `RegisterTool(name, schema, handler)` | `registerTool(name, schema, handler)` |
| Event loop | `Run()` (blocking) | `run()` (blocking) |
| Error handling | Return error | Throw exception |
| Result format | `map[string]any` | `array` |
| Schema | Go types | PHP arrays |
| Thread safety | Single-threaded | Single-threaded |