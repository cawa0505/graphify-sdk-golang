package graphify

import (
	"encoding/json"
	"hash/crc32"
	"path/filepath"
	"time"

	"github.com/cawa0505/graphify-sdk-go/types"
)

// Client is the main Graphify Go SDK client interface.
//
// Provides access to all graphify-mcp tools over Stdio/JSON-RPC.
// Auto-derives workspace_key from the project path.
// Lazy initialization: the transport subprocess spawns on the first request.
type Client struct {
	transport     *Transport
	workspaceKey  string
	workspaceName string
	projectPath   string
}

// ClientOption configures the Graphify client.
type ClientOption func(*Client)

// WithBinaryPath sets the path to the graphify-mcp binary.
func WithBinaryPath(path string) ClientOption {
	return func(c *Client) {
		c.transport = NewTransport(path, 30*time.Second, c.projectPath)
	}
}

// WithTimeout sets the I/O timeout for MCP requests.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.transport = NewTransport("graphify-mcp", timeout, c.projectPath)
	}
}

// NewClient creates a new Graphify client.
//
//   - projectPath: project root path (auto-detects from cwd if empty)
//   - opts: optional configuration options
func NewClient(projectPath string, opts ...ClientOption) *Client {
	if projectPath == "" {
		projectPath = "."
	}
	// Resolve to absolute path
	if abs, err := filepath.Abs(projectPath); err == nil {
		projectPath = abs
	}

	c := &Client{
		projectPath:   projectPath,
		workspaceKey:  deriveWorkspaceKey(projectPath),
		workspaceName: filepath.Base(projectPath),
		transport:     NewTransport("graphify-mcp", 30*time.Second, projectPath),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// deriveWorkspaceKey creates a stable workspace key from a project path.
// Uses crc32 IEEE for cross-platform stability (mirrors the PHP SDK).
func deriveWorkspaceKey(projectPath string) string {
	hash := crc32.ChecksumIEEE([]byte(projectPath))
	return intToHex(int64(hash))
}

func intToHex(n int64) string {
	const hexChars = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	result := make([]byte, 0, 8)
	for n > 0 {
		result = append(result, hexChars[n&0xf])
		n >>= 4
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

// ─── Workspace ────────────────────────────────────────────────────────────

// WorkspaceContext returns the current workspace context.
func (c *Client) WorkspaceContext() types.WorkspaceContext {
	return types.WorkspaceContext{
		WorkspaceKey:  c.workspaceKey,
		WorkspaceName: c.workspaceName,
		RootPath:      c.projectPath,
		Timestamp:     time.Now().Unix(),
	}
}

// ProjectPath returns the project root path.
func (c *Client) ProjectPath() string {
	return c.projectPath
}

// ─── Core Graph Tools ─────────────────────────────────────────────────────

// GraphSummary returns high-level topology metrics.
func (c *Client) GraphSummary() (*types.GraphSummary, error) {
	result, err := c.call("graphify_graph_summary", nil)
	if err != nil {
		return nil, err
	}
	return mapTo[types.GraphSummary](result)
}

// QueryGraph performs a BFS traversal by question.
func (c *Client) QueryGraph(question string) (*types.GraphOutput, error) {
	result, err := c.call("graphify_graph_query", map[string]any{
		"question": question,
	})
	if err != nil {
		return nil, err
	}
	return mapTo[types.GraphOutput](result)
}

// QueryNode queries a node by ID with optional depth.
func (c *Client) QueryNode(nodeID string, depth ...int) (*types.GraphOutput, error) {
	args := map[string]any{"node_id": nodeID}
	if len(depth) > 0 && depth[0] > 0 {
		args["depth"] = depth[0]
	}
	result, err := c.call("graphify_graph_query_node", args)
	if err != nil {
		return nil, err
	}
	return mapTo[types.GraphOutput](result)
}

// TracePath finds the shortest path between two nodes.
func (c *Client) TracePath(from, to string) ([]string, error) {
	result, err := c.call("graphify_graph_trace_path", map[string]any{
		"from": from,
		"to":   to,
	})
	if err != nil {
		return nil, err
	}
	path, _ := result["path"].([]any)
	strs := make([]string, len(path))
	for i, v := range path {
		strs[i], _ = v.(string)
	}
	return strs, nil
}

// ReindexFile reindexes a file into the graph.
func (c *Client) ReindexFile(filePath string) (*types.ReindexResult, error) {
	result, err := c.call("graphify_graph_reindex", map[string]any{
		"file_path": filePath,
	})
	if err != nil {
		return nil, err
	}
	return mapTo[types.ReindexResult](result)
}

// ─── Memory Query ─────────────────────────────────────────────────────────

// MemoryQuery performs a semantic memory search.
func (c *Client) MemoryQuery(query string, limit ...int) (*types.MemoryQueryResult, error) {
	args := map[string]any{
		"workspace_key": c.workspaceKey,
		"query":         query,
	}
	if len(limit) > 0 && limit[0] > 0 {
		args["limit"] = limit[0]
	}
	result, err := c.call("graphify_memory_query", args)
	if err != nil {
		return nil, err
	}
	return mapTo[types.MemoryQueryResult](result)
}

// ─── Relay / Handoff Tools ────────────────────────────────────────────────

// RelayInit initializes a relay session.
func (c *Client) RelayInit(projectContext string, kind ...string) (map[string]any, error) {
	args := map[string]any{"project_context": projectContext}
	if len(kind) > 0 {
		args["kind"] = kind[0]
	}
	return c.call("graphify_relay_init", args)
}

// RelaySave saves session state.
func (c *Client) RelaySave(params map[string]any) (map[string]any, error) {
	return c.call("graphify_relay_save", params)
}

// RelayClose closes a relay session.
func (c *Client) RelayClose(repo, next string) (map[string]any, error) {
	return c.call("graphify_relay_close", map[string]any{
		"repo": repo,
		"next": next,
	})
}

// RelaySwitch switches to another repo.
func (c *Client) RelaySwitch(repo string, kind ...string) (map[string]any, error) {
	args := map[string]any{"repo": repo}
	if len(kind) > 0 {
		args["kind"] = kind[0]
	}
	return c.call("graphify_relay_switch", args)
}

// RelayResume resumes a relay session.
func (c *Client) RelayResume(repo string, kind ...string) (map[string]any, error) {
	args := map[string]any{"repo": repo}
	if len(kind) > 0 {
		args["kind"] = kind[0]
	}
	return c.call("graphify_relay_resume", args)
}

// RelayStatus returns relay status summary.
func (c *Client) RelayStatus() (*types.RelayStatus, error) {
	result, err := c.call("graphify_relay_status", nil)
	if err != nil {
		return nil, err
	}
	return mapTo[types.RelayStatus](result)
}

// RelayAdd ingests a TODO/handoff doc into relay.
func (c *Client) RelayAdd(file, repo string) (map[string]any, error) {
	return c.call("graphify_relay_add", map[string]any{
		"file": file,
		"repo": repo,
	})
}

// ─── OpenDoc Tools ────────────────────────────────────────────────────────

// OpenDocIndex indexes all spec blocks in the workspace.
func (c *Client) OpenDocIndex(docPaths ...string) (map[string]any, error) {
	args := map[string]any{}
	if len(docPaths) > 0 {
		args["doc_paths"] = docPaths
	}
	return c.call("graphify_opendoc_index", args)
}

// OpenDocGetContext returns spec blocks documenting a symbol.
func (c *Client) OpenDocGetContext(symbol string) (map[string]any, error) {
	return c.call("graphify_opendoc_get_context", map[string]any{
		"symbol": symbol,
	})
}

// OpenDocAuditDrift audits doc-side drift.
func (c *Client) OpenDocAuditDrift() (map[string]any, error) {
	return c.call("graphify_opendoc_audit_drift", nil)
}

// ─── Review Tools ─────────────────────────────────────────────────────────

// ReviewIngest imports a CRG review payload.
func (c *Client) ReviewIngest(payload string) (map[string]any, error) {
	return c.call("graphify_review_ingest", map[string]any{
		"payload": payload,
	})
}

// ReviewGetContext queries unresolved reviews for a node.
func (c *Client) ReviewGetContext(node string) (map[string]any, error) {
	return c.call("graphify_review_get_context", map[string]any{
		"node": node,
	})
}

// ReviewResolve marks a review as resolved.
func (c *Client) ReviewResolve(reviewID, reason string) (map[string]any, error) {
	return c.call("graphify_review_resolve", map[string]any{
		"review_id": reviewID,
		"reason":    reason,
	})
}

// ReviewSearchCrg searches CRG for changed functions.
func (c *Client) ReviewSearchCrg(base ...string) (map[string]any, error) {
	args := map[string]any{}
	if len(base) > 0 {
		args["base"] = base[0]
	}
	return c.call("graphify_review_search_crg", args)
}

// ─── Telemetry Tools ──────────────────────────────────────────────────────

// TelemetryIngest imports telemetry metrics.
func (c *Client) TelemetryIngest(source string, pathOrDracoParams ...string) (map[string]any, error) {
	args := map[string]any{"source": source}
	if len(pathOrDracoParams) > 0 {
		args["path_or_draco_params"] = pathOrDracoParams[0]
	}
	return c.call("graphify_telemetry_ingest", args)
}

// TelemetryGetContext queries telemetry bindings for a node.
func (c *Client) TelemetryGetContext(node string, includeImpactRadius ...bool) (map[string]any, error) {
	args := map[string]any{"node": node}
	if len(includeImpactRadius) > 0 && includeImpactRadius[0] {
		args["include_impact_radius"] = true
	}
	return c.call("graphify_telemetry_get_context", args)
}

// ─── Coverage Tools ───────────────────────────────────────────────────────

// CoverageIngest imports coverage data.
func (c *Client) CoverageIngest(format, data string) (map[string]any, error) {
	return c.call("graphify_coverage_ingest", map[string]any{
		"format": format,
		"data":   data,
	})
}

// CoverageGetContext queries coverage for a node.
func (c *Client) CoverageGetContext(node string) (*types.CoverageResult, error) {
	result, err := c.call("graphify_coverage_get_context", map[string]any{
		"node": node,
	})
	if err != nil {
		return nil, err
	}
	return mapTo[types.CoverageResult](result)
}

// CoverageBlindspots lists low-coverage nodes.
func (c *Client) CoverageBlindspots() (map[string]any, error) {
	return c.call("graphify_coverage_blindspots", nil)
}

// ─── Plugin Gateway ───────────────────────────────────────────────────────

// PluginNotify broadcasts a graph update notification to all plugins.
func (c *Client) PluginNotify(kind string) (map[string]any, error) {
	return c.call("graphify_plugin_notify", map[string]any{
		"kind": kind,
	})
}

// ─── Lifecycle ────────────────────────────────────────────────────────────

// Start explicitly starts the transport. Auto-starts on first request.
func (c *Client) Start() error {
	return c.transport.Start()
}

// Stop stops the transport and cleans up.
func (c *Client) Stop() {
	c.transport.Stop()
}

// IsRunning reports whether the transport subprocess is running.
func (c *Client) IsRunning() bool {
	return c.transport.IsRunning()
}

// ─── Internal ─────────────────────────────────────────────────────────────

// call is a low-level MCP tool call.
func (c *Client) call(method string, args map[string]any) (map[string]any, error) {
	if args == nil {
		args = map[string]any{}
	}
	return c.transport.Call(method, args)
}

// mapTo converts a raw map to a typed struct via JSON round-trip.
func mapTo[T any](data map[string]any) (*T, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, NewProtocolError("marshal intermediate: %w", err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, NewProtocolError("unmarshal to %T: %w", out, err)
	}
	return &out, nil
}