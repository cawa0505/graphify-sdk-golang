// Package plugin provides the JSON-RPC stdio host for Graphify Plugin SDK.
//
// Runs as a subprocess (IPC mode), receives JSON-RPC requests on stdin,
// dispatches to registered tool handlers, and writes responses to stdout.
//
// Protocol: MCP JSON-RPC (subset)
//   - initialize          → returns tools + capabilities
//   - tools/list          → returns registered tools
//   - tools/call          → dispatches to handler
//   - notifications/*     → silently accepted (no response)
//
// Usage:
//
//	host := plugin.NewHost()
//	host.RegisterTool("analyze_schema", plugin.ToolSchema{
//	    Description: "Analyze Laravel database schema",
//	    InputSchema: map[string]any{
//	        "type": "object",
//	        "properties": map[string]any{...},
//	    },
//	}, func(args map[string]any) (map[string]any, error) {
//	    return map[string]any{"columns": [...]}, nil
//	})
//	host.Run() // blocks, reads stdin forever
package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ToolSchema describes a tool that this plugin provides.
type ToolSchema struct {
	Description string
	InputSchema map[string]any
}

// ToolHandler is a function that handles a tool call.
type ToolHandler func(args map[string]any) (map[string]any, error)

// jsonRPCRequest represents a JSON-RPC request received by the plugin host.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC response sent by the plugin host.
type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// toolsCallParams is the params structure for a tools/call request.
type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Host manages the JSON-RPC stdio listener for a Graphify plugin.
type Host struct {
	tools   []toolEntry
	running bool
}

type toolEntry struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     ToolHandler
}

// NewHost creates a new plugin host.
func NewHost() *Host {
	return &Host{}
}

// RegisterTool registers a tool that this plugin provides.
func (h *Host) RegisterTool(name string, schema ToolSchema, handler ToolHandler) *Host {
	h.tools = append(h.tools, toolEntry{
		Name:        name,
		Description: schema.Description,
		InputSchema: schema.InputSchema,
		Handler:     handler,
	})
	return h
}

// Run starts the JSON-RPC stdio listener (blocking).
//
// Reads newline-delimited JSON from stdin, writes responses to stdout.
// Exits cleanly when stdin closes (parent process terminated the pipe).
func (h *Host) Run() {
	h.running = true
	scanner := bufio.NewScanner(os.Stdin)

	for h.running && scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue // skip malformed lines
		}

		if req.Method == "" {
			continue
		}

		// Notifications (no id) — no response expected
		if req.ID == nil {
			continue
		}

		result, err := h.dispatch(req.Method, req.Params)
		if err != nil {
			h.sendError(req.ID, -32603, err.Error())
		} else {
			h.sendResponse(req.ID, result)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "plugin host: read error: %v\n", err)
	}
	os.Exit(0)
}

// Stop gracefully stops the listener loop.
func (h *Host) Stop() {
	h.running = false
}

// dispatch routes a method to the appropriate handler.
func (h *Host) dispatch(method string, rawParams json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return h.handleInitialize(), nil
	case "tools/list":
		return h.handleToolsList(), nil
	case "tools/call":
		return h.handleToolsCall(rawParams)
	default:
		// Notifications are handled before dispatch, so unknown methods are errors
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// handleInitialize returns protocol version, capabilities, and tools.
func (h *Host) handleInitialize() map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": struct{}{},
		},
		"tools": h.buildToolList(),
	}
}

// handleToolsList returns registered tools.
func (h *Host) handleToolsList() map[string]any {
	return map[string]any{
		"tools": h.buildToolList(),
	}
}

// handleToolsCall dispatches to a registered handler.
func (h *Host) handleToolsCall(rawParams json.RawMessage) (map[string]any, error) {
	var params toolsCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	for _, tool := range h.tools {
		if tool.Name == params.Name {
			result, err := tool.Handler(params.Arguments)
			if err != nil {
				return nil, err
			}

			resultJSON, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("marshal result: %w", err)
			}

			return map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": string(resultJSON),
					},
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("tool not found: %s", params.Name)
}

// buildToolList builds the tools array for initialize / tools/list responses.
func (h *Host) buildToolList() []map[string]any {
	list := make([]map[string]any, 0, len(h.tools))
	for _, t := range h.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		list = append(list, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": schema,
		})
	}
	return list
}

// ─── I/O helpers ──────────────────────────────────────────────────────────

// sendResponse writes a JSON-RPC success response to stdout.
func (h *Host) sendResponse(id json.RawMessage, result any) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

// sendError writes a JSON-RPC error response to stdout.
func (h *Host) sendError(id json.RawMessage, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}