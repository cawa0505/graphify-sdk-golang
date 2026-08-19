package graphify

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"
	"sync"
	"time"
)

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  mcpCallArgs `json:"params"`
}

// mcpCallArgs wraps the MCP tools/call parameters.
type mcpCallArgs struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPContentItem represents a content item in an MCP tool result.
type MCPContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPResult is the decoded result of an MCP tools/call response.
type MCPResult struct {
	Content []MCPContentItem    `json:"content,omitempty"`
	Extra   map[string]json.RawMessage `json:"-"`
}

// Transport manages the lifecycle of a graphify subprocess and provides
// JSON-RPC 2.0 request/response over stdin/stdout pipes.
//
// Zero external dependencies: uses only the Go standard library (os/exec, encoding/json).
// Lazy initialization: the subprocess spawns on the first request.
type Transport struct {
	binaryPath string
	timeout    time.Duration
	cwd        string

	mu        sync.Mutex
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	started   bool
	requestID int
}

// NewTransport creates a new MCP transport.
//
//   - binaryPath: path to the graphify binary (default: "graphify")
//   - timeout: I/O timeout (default: 30s)
//   - cwd: working directory for the subprocess (optional)
func NewTransport(binaryPath string, timeout time.Duration, cwd string) *Transport {
	if binaryPath == "" {
		binaryPath = "graphify"
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Transport{
		binaryPath: binaryPath,
		timeout:    timeout,
		cwd:        cwd,
	}
}

// Start spawns the graphify subprocess. Idempotent — safe to call multiple times.
func (t *Transport) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return nil
	}

	cmd := exec.Command(t.binaryPath)
	if t.cwd != "" {
		cmd.Dir = t.cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return NewTransportError("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return NewTransportError("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return NewTransportError("spawn graphify: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = bufio.NewScanner(stdout)
	t.started = true
	return nil
}

// Call sends a JSON-RPC 2.0 request and returns the decoded result.
func (t *Transport) Call(method string, args map[string]any) (map[string]any, error) {
	if err := t.Start(); err != nil {
		return nil, err
	}

	t.mu.Lock()
	t.requestID++
	id := t.requestID
	t.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: mcpCallArgs{
			Name:      method,
			Arguments: args,
		},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, NewProtocolError("encode request: %w", err)
	}

	// Write request
	reqData = append(reqData, '\n')
	if _, err := t.stdin.Write(reqData); err != nil {
		return nil, NewTransportError("write stdin: %w", err)
	}

	// Read response with timeout
	type result struct {
		data map[string]any
		err  error
	}

	done := make(chan result, 1)
	go func() {
		data, err := t.readResponse()
		done <- result{data, err}
	}()

	select {
	case r := <-done:
		return r.data, r.err
	case <-time.After(t.timeout):
		t.stopLocked()
		return nil, NewTransportError("timeout after %v", t.timeout)
	}
}

// readResponse reads and parses a JSON-RPC response from stdout.
func (t *Transport) readResponse() (map[string]any, error) {
	t.mu.Lock()
	scanner := t.stdout
	t.mu.Unlock()

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, NewTransportError("read stdout: %w", err)
		}
		return nil, NewTransportError("stdout closed (process exited)")
	}

	line := scanner.Text()
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, NewProtocolError("parse response: %w", err)
	}

	if resp.Error != nil {
		return nil, NewEngineError(resp.Error.Code, resp.Error.Message, nil)
	}

	// Parse result
	var mcpResult MCPResult
	if err := json.Unmarshal(resp.Result, &mcpResult); err != nil {
		return nil, NewProtocolError("parse result: %w", err)
	}

	// Extract text from MCP content items and try to parse as JSON
	if len(mcpResult.Content) > 0 {
		var combined string
		for _, c := range mcpResult.Content {
			combined += c.Text
		}

		var parsed map[string]any
		if err := json.Unmarshal([]byte(combined), &parsed); err == nil {
			return parsed, nil
		}

		return map[string]any{"text": combined}, nil
	}

	// Fallback: try to parse result as a plain object
	var resultMap map[string]any
	if err := json.Unmarshal(resp.Result, &resultMap); err == nil {
		return resultMap, nil
	}

	return map[string]any{}, nil
}

// Stop terminates the graphify subprocess and cleans up pipes.
func (t *Transport) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopLocked()
}

func (t *Transport) stopLocked() {
	if !t.started {
		return
	}

	if t.stdin != nil {
		t.stdin.Close()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}

	t.cmd = nil
	t.stdin = nil
	t.stdout = nil
	t.started = false
}

// IsRunning reports whether the subprocess is currently running.
func (t *Transport) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.started && t.cmd != nil && t.cmd.Process != nil
}

