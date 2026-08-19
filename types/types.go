// Package types provides data transfer objects (DTOs) for the Graphify SDK.
//
// These types mirror the Rust structs in graphify_core and provide
// JSON deserialization from graphify tool responses.
package types

// ─── FileType ─────────────────────────────────────────────────────────────

// FileType represents the classification of a file in the knowledge graph.
type FileType string

const (
	FileTypeCode      FileType = "code"
	FileTypeDocument  FileType = "document"
	FileTypePaper     FileType = "paper"
	FileTypeImage     FileType = "image"
	FileTypeRationale FileType = "rationale"
	FileTypeConcept   FileType = "concept"
)

// ─── NodeId ───────────────────────────────────────────────────────────────

// NodeId is a value object representing a node ID in the Graphify knowledge graph.
//
// Format: "./path/to/file:kind:Name" (e.g., "./src/lib.rs:function:MyFunction")
type NodeId struct {
	ID string `json:"id"`
}

// Parse splits a NodeId into its components: file, kind, name.
func (n NodeId) Parse() (file, kind, name string) {
	// Simple split on first two colons
	parts := splitAt(n.ID, ':', 3)
	if len(parts) > 0 {
		file = parts[0]
	}
	if len(parts) > 1 {
		kind = parts[1]
	}
	if len(parts) > 2 {
		name = parts[2]
	}
	return
}

func splitAt(s string, sep byte, n int) []string {
	var result []string
	start := 0
	for i := 0; i < len(s) && len(result) < n-1; i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// ─── Node ─────────────────────────────────────────────────────────────────

// Node represents a node in the Graphify knowledge graph.
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

// ─── Edge ─────────────────────────────────────────────────────────────────

// Edge represents a relationship between two nodes in the knowledge graph.
type Edge struct {
	Source         string  `json:"source"`
	Target         string  `json:"target"`
	Relation       string  `json:"relation"`
	SourceFile     string  `json:"source_file"`
	Confidence     string  `json:"confidence"`
	SourceLocation string  `json:"source_location"`
	Description    *string `json:"description,omitempty"`
}

// ─── GraphMetadata ────────────────────────────────────────────────────────

// GraphMetadata contains metadata about a Graphify knowledge graph.
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

// ─── GraphOutput ──────────────────────────────────────────────────────────

// GraphOutput is the complete result of a graph traversal query.
type GraphOutput struct {
	Nodes    []Node        `json:"nodes"`
	Edges    []Edge        `json:"edges"`
	Metadata GraphMetadata `json:"metadata"`
}

// ─── GraphSummary ─────────────────────────────────────────────────────────

// GraphSummary is the result of a graphify_graph_summary call.
type GraphSummary struct {
	TotalNodes int      `json:"total_nodes"`
	TotalEdges int      `json:"total_edges"`
	Languages  []string `json:"languages"`
}

// ─── WorkspaceContext ─────────────────────────────────────────────────────

// WorkspaceContext holds workspace identification information.
type WorkspaceContext struct {
	WorkspaceKey  string `json:"workspace_key"`
	WorkspaceName string `json:"workspace_name"`
	RootPath      string `json:"root_path"`
	Timestamp     int64  `json:"timestamp"`
}

// ─── MemoryQueryResult ────────────────────────────────────────────────────

// MemoryQueryResult is the result of a semantic memory query.
type MemoryQueryResult struct {
	Status string   `json:"status"`
	Nodes  []Node   `json:"nodes,omitempty"`
	Raw    []map[string]any `json:"-"` // raw node data before parsing
}

// IsFound returns true when the memory query found results.
func (m *MemoryQueryResult) IsFound() bool {
	return m.Status == "found"
}

// ─── CoverageResult ───────────────────────────────────────────────────────

// CoverageResult contains coverage information for a node.
type CoverageResult struct {
	Node         string  `json:"node"`
	CoveredLines int     `json:"covered_lines"`
	TotalLines   int     `json:"total_lines"`
	LineRate     float64 `json:"line_rate"`
}

// Percentage returns the coverage ratio as a percentage (0-100).
func (c *CoverageResult) Percentage() float64 {
	return c.LineRate * 100
}

// ─── ReindexResult ────────────────────────────────────────────────────────

// ReindexResult is the result of a reindex operation.
type ReindexResult struct {
	Status     string `json:"status"`
	TotalNodes int    `json:"total_nodes"`
	TotalEdges int    `json:"total_edges"`
}

// IsSuccess returns true when the reindex was successful.
func (r *ReindexResult) IsSuccess() bool {
	return r.Status == "success"
}

// ─── RelayStatus ──────────────────────────────────────────────────────────

// RelayStatus is the result of a relay_status call.
type RelayStatus struct {
	Repos       map[string]any `json:"repos"`
	ActiveBaton *string        `json:"active_baton,omitempty"`
	SpecDrift   *string        `json:"spec_drift,omitempty"`
	LastUpdate  *string        `json:"last_update,omitempty"`
}

// ─── ReviewFinding ────────────────────────────────────────────────────────

// ReviewFinding represents a code review finding.
type ReviewFinding struct {
	ReviewID         string   `json:"review_id"`
	AffectedSymbols  []string `json:"affected_symbols"`
	FindingSeverity  string   `json:"finding_severity"`
	ResolutionStatus string   `json:"resolution_status"`
	ReviewComment    string   `json:"review_comment"`
	GitCommitSHA     *string  `json:"git_commit_sha,omitempty"`
}

// IsResolved returns true when the finding has been resolved.
func (r *ReviewFinding) IsResolved() bool {
	return r.ResolutionStatus == "resolved"
}

// ─── TelemetryBinding ─────────────────────────────────────────────────────

// TelemetryBinding contains telemetry metrics for a node.
type TelemetryBinding struct {
	Node         string           `json:"node"`
	P99Latency   float64          `json:"p99_latency"`
	AllocBytes   float64          `json:"alloc_bytes"`
	CallRate     float64          `json:"call_rate"`
	IsHotspot    bool             `json:"is_hotspot"`
	ImpactRadius []map[string]any `json:"impact_radius,omitempty"`
}

// ─── Helpers ──────────────────────────────────────────────────────────────

// NodesFromRaw converts a slice of raw maps to Node values.
func NodesFromRaw(raw []map[string]any) []Node {
	nodes := make([]Node, 0, len(raw))
	for _, r := range raw {
		nodes = append(nodes, NodeFromMap(r))
	}
	return nodes
}

// NodeFromMap constructs a Node from a raw map.
func NodeFromMap(m map[string]any) Node {
	n := Node{
		ID:         stringVal(m["id"]),
		Label:      stringVal(m["label"]),
		FileType:   stringVal(m["file_type"]),
		Kind:       stringVal(m["kind"]),
		Language:   stringVal(m["language"]),
		SourceFile: stringVal(m["source_file"]),
		StartLine:  intVal(m["start_line"]),
		EndLine:    intVal(m["end_line"]),
	}
	if v, ok := m["doc_comment"].(string); ok && v != "" {
		n.DocComment = &v
	}
	if v, ok := m["description"].(string); ok && v != "" {
		n.Description = &v
	}
	if v, ok := m["metadata"].(map[string]any); ok {
		n.Metadata = v
	}
	return n
}

func stringVal(v any) string {
	s, _ := v.(string)
	return s
}

func intVal(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return 0
	}
}