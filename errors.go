// Package graphify provides the Graphify Go SDK — access Graphify's knowledge
// graph capabilities over the MCP (Model Context Protocol) via Stdio/JSON-RPC.
package graphify

import "fmt"

// GraphifyError is the base error type for all Graphify SDK errors.
type GraphifyError struct {
	Msg string
}

func (e *GraphifyError) Error() string { return e.Msg }

// TransportError is raised when there is an I/O or process management error
// with the MCP transport.
type TransportError struct{ GraphifyError }

func NewTransportError(format string, a ...any) *TransportError {
	return &TransportError{GraphifyError{Msg: fmt.Sprintf("[Transport] "+format, a...)}}
}

// ProtocolError is raised when there is a JSON-RPC protocol error.
type ProtocolError struct{ GraphifyError }

func NewProtocolError(format string, a ...any) *ProtocolError {
	return &ProtocolError{GraphifyError{Msg: fmt.Sprintf("[Protocol] "+format, a...)}}
}

// EngineError is raised when graphify returns an engine-level error.
type EngineError struct {
	GraphifyError
	Code      int
	ErrorData map[string]any
}

func NewEngineError(code int, message string, data map[string]any) *EngineError {
	return &EngineError{
		GraphifyError: GraphifyError{Msg: fmt.Sprintf("[Engine] %s", message)},
		Code:          code,
		ErrorData:     data,
	}
}