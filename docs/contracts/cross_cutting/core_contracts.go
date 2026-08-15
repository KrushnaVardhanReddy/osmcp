// osmcp — Core Interface Contracts (Cross-cutting)
// These interfaces are defined BEFORE any implementation is written.
// No implementation file may exist before its corresponding contract is approved.
//
// Spec references:
//   docs/specs/cross_cutting/01_response_envelope_spec.md
//   docs/specs/cross_cutting/02_policy_engine_spec.md

package contracts

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// ─────────────────────────────────────────────────────────────────────────────
// Response Envelope
// ─────────────────────────────────────────────────────────────────────────────

// Envelope is the uniform response shape returned by every tool call.
// Spec: docs/specs/cross_cutting/01_response_envelope_spec.md
type Envelope struct {
	Version string      `json:"version"` // Always "1" in v1
	OK      bool        `json:"ok"`
	Tool    string      `json:"tool"`
	Data    interface{} `json:"data"`  // Tool-specific payload; null on failure
	Meta    Meta        `json:"meta"`
	Error   *Error      `json:"error"` // null on success
}

// Meta holds cross-cutting metadata present in every response.
type Meta struct {
	ExecutionTimeMs int64 `json:"execution_time_ms"`
	Truncated       bool  `json:"truncated"`
}

// Error is the structured error object. Code is always a stable enum value.
type Error struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable"`
}

// ErrorCode is the stable enum of all possible error conditions.
// No new values may be added without a minor version bump.
type ErrorCode string

const (
	ErrPolicyDenied   ErrorCode = "POLICY_DENIED"
	ErrInvalidArgs    ErrorCode = "INVALID_ARGS"
	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrTimeout        ErrorCode = "TIMEOUT"
	ErrOutputTooLarge ErrorCode = "OUTPUT_TOO_LARGE"
	ErrExecFailed     ErrorCode = "EXEC_FAILED"
)

// EnvelopeBuilder is the ONLY way to construct response envelopes.
// Tools must not build JSON responses directly.
// Spec: docs/specs/cross_cutting/01_response_envelope_spec.md §7
type EnvelopeBuilder interface {
	// Success builds an ok=true envelope with the given tool-specific data.
	Success(tool string, data interface{}, meta Meta) Envelope

	// Failure builds an ok=false envelope with a structured error.
	Failure(tool string, code ErrorCode, message string, retryable bool, meta Meta) Envelope
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy Engine
// ─────────────────────────────────────────────────────────────────────────────

// PolicyEngine is the central safety gate. Every tool call passes through Evaluate
// before any execution occurs.
// Spec: docs/specs/cross_cutting/02_policy_engine_spec.md
type PolicyEngine interface {
	// Evaluate checks whether the given tool call is permitted by the active policy.
	//
	// toolName  — the MCP tool name (e.g. "grep")
	// pathArgs  — all path-type arguments from the tool call (symlinks resolved internally)
	// isMutating — true if this tool call modifies the filesystem or git state
	//
	// Returns nil on ALLOW; returns a *PolicyError on DENY.
	// Evaluate also writes one audit log entry regardless of decision.
	Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error

	// IsToolVisible returns true if the given tool name is in the active policy's
	// AllowedTools list. Used to filter the tools/list MCP response.
	IsToolVisible(toolName string) bool

	// Limits returns the resource limits from the active policy (timeout, output cap, etc.)
	Limits() PolicyLimits

	// AllowedRoot returns the configured allowed root path.
	AllowedRoot() string

	// RunScriptConfig returns the run_script specific configuration
	RunScriptConfig() RunScriptConfig
}

// RunScriptConfig holds the run_script specific configuration from the active policy.
type RunScriptConfig struct {
	BlockedBinaries []string
	AllowNetwork    bool
}

// PolicyLimits holds the resource limits enforced by the Execution Engine.
type PolicyLimits struct {
	TimeoutMs      int64 // Per-call wall-clock limit
	MaxOutputBytes int64 // Maximum combined stdout+stderr
	MaxMatches     int   // Maximum match results (grep, find)
}

// PolicyError is returned by PolicyEngine.Evaluate on a DENY decision.
// It maps directly to the POLICY_DENIED error code in the response envelope.
type PolicyError struct {
	Reason string // Human-readable denial reason
}

func (e *PolicyError) Error() string { return e.Reason }



// ─────────────────────────────────────────────────────────────────────────────
// Tool
// ─────────────────────────────────────────────────────────────────────────────

// Tool is the interface every typed tool must implement.
// Tools are registered with the MCP server via the ToolRegistry.
type Tool interface {
	// Name returns the MCP tool name. Must be stable — changing it is a breaking change.
	Name() string

	// Description returns the human-readable tool description shown in tools/list.
	Description() string

	// IsMutating returns true if this tool modifies filesystem or git state.
	// Used by PolicyEngine.Evaluate.
	IsMutating() bool

	// RegisterMCP defines the JSON-RPC schema and registers the tool execution handler with the MCP server.
	RegisterMCP(s *server.MCPServer)
}

// ToolRegistry manages the set of tools registered with the MCP server.
// It filters visible tools based on the active policy.
type ToolRegistry interface {
	// Register adds a tool to the registry. Called at startup before the server runs.
	Register(tool Tool)

	// VisibleTools returns all tools that are permitted by the active policy.
	// This is the set returned to agents in the MCP tools/list response.
	VisibleTools() []Tool
}

// ─────────────────────────────────────────────────────────────────────────────
// Audit Logger
// ─────────────────────────────────────────────────────────────────────────────

// AuditLogger writes one structured NDJSON log entry per tool call.
// Logging is non-optional and always active.
// Spec: docs/specs/cross_cutting/02_policy_engine_spec.md §7
type AuditLogger interface {
	// Log writes a single audit entry. Called by the Policy Engine after every
	// Evaluate() call, regardless of ALLOW or DENY.
	Log(entry AuditEntry)
}

// AuditEntry is the structured audit log record.
type AuditEntry struct {
	Timestamp      time.Time `json:"ts"`
	CallID         string    `json:"call_id"`
	Tool           string    `json:"tool"`
	PathArgs       []string  `json:"path_args"`
	PolicyDecision string    `json:"policy_decision"` // "ALLOW" or "DENIED"
	DenialCode     string    `json:"denial_code,omitempty"`
	DurationMs     int64     `json:"duration_ms"`
	OK             bool      `json:"ok"`
}
