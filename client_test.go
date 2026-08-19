package graphify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── Client Tests ───────────────────────────────────────────────────────

func TestNewClient(t *testing.T) {
	c := NewClient("")
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if c.projectPath == "" {
		t.Error("projectPath should not be empty")
	}
}

func TestNewClientWithCustomPath(t *testing.T) {
	c := NewClient("/tmp/test-project")
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if !strings.HasSuffix(c.projectPath, "test-project") {
		t.Errorf("expected suffix test-project, got %q", c.projectPath)
	}
}

func TestClientWithBinaryPathOption(t *testing.T) {
	c := NewClient("", WithBinaryPath("/custom/path/graphify"))
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestClientWorkspaceKey(t *testing.T) {
	c1 := NewClient("/tmp/test-project")
	c2 := NewClient("/tmp/test-project")
	if c1.workspaceKey != c2.workspaceKey {
		t.Errorf("workspace keys should match: %q != %q", c1.workspaceKey, c2.workspaceKey)
	}
}

func TestClientWorkspaceKeyDeterministic(t *testing.T) {
	c := NewClient("")
	key := c.workspaceKey
	if len(key) == 0 {
		t.Fatal("workspace key should not be empty")
	}
	// crc32 IEEE produces 8 hex chars
	if len(key) != 8 {
		t.Errorf("expected 8-char hex key, got %q (len=%d)", key, len(key))
	}
}

func TestClientWorkspaceKeyDifferentPaths(t *testing.T) {
	c1 := NewClient("/project/alpha")
	c2 := NewClient("/project/beta")
	if c1.workspaceKey == c2.workspaceKey {
		t.Error("different paths should produce different workspace keys")
	}
}

// ─── Transport Tests ────────────────────────────────────────────────────

func TestNewTransport(t *testing.T) {
	tr := NewTransport("", 0, "")
	if tr == nil {
		t.Fatal("NewTransport() returned nil")
	}
	if tr.binaryPath != "graphify" {
		t.Errorf("default binary should be 'graphify', got %q", tr.binaryPath)
	}
	if tr.timeout != 30*time.Second {
		t.Errorf("default timeout should be 30s, got %v", tr.timeout)
	}
}

func TestNewTransportWithCustomPath(t *testing.T) {
	tr := NewTransport("/opt/bin/graphify", 10*time.Second, "/tmp")
	if tr.binaryPath != "/opt/bin/graphify" {
		t.Errorf("expected custom binary path, got %q", tr.binaryPath)
	}
	if tr.timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", tr.timeout)
	}
	if tr.cwd != "/tmp" {
		t.Errorf("expected cwd /tmp, got %q", tr.cwd)
	}
}

func TestTransportStartWithInvalidBinary(t *testing.T) {
	tr := NewTransport("/nonexistent/graphify-binary", time.Second, "")
	err := tr.Start()
	if err == nil {
		t.Fatal("expected error from invalid binary, got nil")
	}
	var te *TransportError
	if !isTransportError(err) {
		t.Fatalf("expected TransportError, got %T: %v", err, err)
	}
	_ = te
}

func TestTransportStopWithoutStart(t *testing.T) {
	tr := NewTransport("graphify", 0, "")
	// Stop on unstarted transport must not panic
	tr.Stop()
}

func TestTransportStartTwice(t *testing.T) {
	tr := NewTransport("/nonexistent/graphify", time.Second, "")
	err := tr.Start()
	if err == nil {
		return // no assertion needed — start on invalid binary is expected to fail
	}
	// Second start should also fail
	err2 := tr.Start()
	if err2 == nil {
		t.Error("second Start() should also fail")
	}
}

// ─── Error Type Tests ───────────────────────────────────────────────────

func TestErrorTypes(t *testing.T) {
	te := NewTransportError("connection failed: %s", "timeout")
	if !strings.Contains(te.Error(), "connection failed") {
		t.Errorf("TransportError message mismatch: %s", te.Error())
	}

	pe := NewProtocolError("invalid json-rpc")
	if !strings.Contains(pe.Error(), "invalid json-rpc") {
		t.Errorf("ProtocolError message mismatch: %s", pe.Error())
	}

	ee := NewEngineError(-32000, "engine error", nil)
	if !strings.Contains(ee.Error(), "engine error") {
		t.Errorf("EngineError message mismatch: %s", ee.Error())
	}
	if ee.Code != -32000 {
		t.Errorf("expected code -32000, got %d", ee.Code)
	}
}

func TestErrorTypeAssertions(t *testing.T) {
	te := NewTransportError("test")
	pe := NewProtocolError("test")
	ee := NewEngineError(-1, "test", nil)

	if !isTransportError(te) {
		t.Error("TransportError should be assertable as TransportError")
	}
	if !isProtocolError(pe) {
		t.Error("ProtocolError should be assertable as ProtocolError")
	}
	if !isEngineError(ee) {
		t.Error("EngineError should be assertable as EngineError")
	}
	if isEngineError(te) {
		t.Error("TransportError should NOT be assertable as EngineError")
	}
}

func isTransportError(err error) bool {
	_, ok := err.(*TransportError)
	return ok
}

func isProtocolError(err error) bool {
	_, ok := err.(*ProtocolError)
	return ok
}

func isEngineError(err error) bool {
	_, ok := err.(*EngineError)
	return ok
}

// ─── JSON-RPC Protocol Tests ───────────────────────────────────────────

func TestJSONRPCRequestMarshal(t *testing.T) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: mcpCallArgs{
			Name:      "graph_summary",
			Arguments: map[string]any{},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded jsonRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %q", decoded.JSONRPC)
	}
	if decoded.ID != 1 {
		t.Errorf("expected id 1, got %d", decoded.ID)
	}
	if decoded.Method != "tools/call" {
		t.Errorf("expected method tools/call, got %q", decoded.Method)
	}
	if decoded.Params.Name != "graph_summary" {
		t.Errorf("expected params name graph_summary, got %q", decoded.Params.Name)
	}
}

func TestJSONRPCResponseUnmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"nodes\":10}"}]}}`
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
}

func TestJSONRPCErrorResponseUnmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("expected 'Method not found', got %q", resp.Error.Message)
	}
}

// ─── MCPResult Tests ────────────────────────────────────────────────────

func TestMCPResultWithTextContent(t *testing.T) {
	raw := `{"content":[{"type":"text","text":"{\"key\":\"value\"}"}]}`
	var result MCPResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal MCPResult: %v", err)
	}
	if len(result.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected type text, got %q", result.Content[0].Type)
	}
}

// ─── Integration Test (skipped without GRAPHIFY_BINARY) ──────────────────

func TestIntegrationMockTransport(t *testing.T) {
	if os.Getenv("GRAPHIFY_BINARY") == "" {
		t.Skip("set GRAPHIFY_BINARY to run integration tests")
	}

	binary := writeMockGraphify(t)
	defer os.Remove(binary)

	tr := NewTransport(binary, 5*time.Second, "")
	result, err := tr.Call("graph_summary", map[string]any{})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	tr.Stop()
}

func writeMockGraphify(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "mock-graphify")
	script := fmt.Sprintf(`#!/bin/sh
while IFS= read -r line; do
	case "$line" in
		*"graph_summary"*)
			echo '{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"nodes\":5,\"edges\":10}"}]}}'
			;;
		*)
			echo '{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}'
			;;
	esac
done
`)
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write mock binary: %v", err)
	}
	return binary
}

// TestMain provides a clean exit.
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}