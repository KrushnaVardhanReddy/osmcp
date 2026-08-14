// osmcp — grep Tool Contract (Phase 1)
// This contract defines the exact input/output types for the grep tool.
// Implementation in internal/tools/grep.go must satisfy this contract exactly.
//
// Spec reference: docs/specs/phase-1/01_grep_tool_spec.md
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be approved first.

package contracts

// ─────────────────────────────────────────────────────────────────────────────
// grep — Input Args
// ─────────────────────────────────────────────────────────────────────────────

// GrepArgs defines the input arguments for the grep tool.
// JSON tags match the MCP tool schema exactly (see spec §3).
// jsonschema tags are used by the MCP SDK for automatic schema generation.
type GrepArgs struct {
	// Pattern is the search term. Treated as a regular expression unless Literal=true.
	// Required. Must not be empty.
	Pattern string `json:"pattern" jsonschema:"required,description=Search pattern. Treated as a regex unless literal=true."`

	// Path is an absolute path to a file or directory to search.
	// Required. Must be within policy.AllowedRoot after symlink resolution.
	Path string `json:"path" jsonschema:"required,description=Absolute path to file or directory."`

	// Recursive searches subdirectories when Path is a directory.
	// Default: true.
	Recursive bool `json:"recursive" jsonschema:"default=true,description=Search subdirectories recursively."`

	// CaseSensitive controls whether the search is case-sensitive.
	// Default: true.
	CaseSensitive bool `json:"case_sensitive" jsonschema:"default=true,description=Case-sensitive search."`

	// Literal treats Pattern as a fixed string, not a regex.
	// Default: false.
	Literal bool `json:"literal" jsonschema:"default=false,description=Treat pattern as a fixed string (not regex)."`

	// ContextLines is the number of lines of context before and after each match.
	// Range: 0–10. Default: 0.
	ContextLines int `json:"context_lines" jsonschema:"default=0,minimum=0,maximum=10,description=Lines of context around each match."`

	// Include is a glob pattern to restrict which files are searched.
	// Example: "*.go". Optional.
	Include string `json:"include,omitempty" jsonschema:"description=Glob to restrict searched files. e.g. '*.go'"`

	// Exclude is a glob pattern to exclude files from the search.
	// Example: "*_test.go". Optional.
	Exclude string `json:"exclude,omitempty" jsonschema:"description=Glob to exclude files from search."`

	// MaxMatches overrides the policy max_matches for this call.
	// Cannot exceed the policy limit. Optional.
	MaxMatches int `json:"max_matches,omitempty" jsonschema:"description=Override policy max_matches for this call."`
}

// ─────────────────────────────────────────────────────────────────────────────
// grep — Output Data
// ─────────────────────────────────────────────────────────────────────────────

// GrepData is the tool-specific payload in the Envelope.Data field on success.
// Spec: docs/specs/phase-1/01_grep_tool_spec.md §5
type GrepData struct {
	// Matches is the ordered list of match results.
	// Always an array — never null. Empty array when no matches found.
	Matches []GrepMatch `json:"matches"`

	// Count is the total number of matches collected.
	// Equals len(Matches) unless meta.Truncated is true.
	Count int `json:"count"`
}

// GrepMatch represents a single line that matched the pattern.
type GrepMatch struct {
	// File is the absolute path to the file containing the match.
	File string `json:"file"`

	// Line is the 1-indexed line number of the match.
	Line int `json:"line"`

	// Text is the matched line content, trimmed of trailing newline.
	Text string `json:"text"`

	// ContextBefore contains lines before the match.
	// Empty slice when context_lines=0.
	ContextBefore []string `json:"context_before"`

	// ContextAfter contains lines after the match.
	// Empty slice when context_lines=0.
	ContextAfter []string `json:"context_after"`
}

// ─────────────────────────────────────────────────────────────────────────────
// grep — Behaviour Invariants (verified by tests)
// ─────────────────────────────────────────────────────────────────────────────

// The following invariants MUST hold for all grep implementations.
// Each invariant has a corresponding unit test in internal/tools/grep_test.go.
//
// INV-GREP-01: No matches → GrepData{Matches: [], Count: 0}, ok=true (not an error)
// INV-GREP-02: Truncated results → meta.Truncated=true, count=len(matches) (not total)
// INV-GREP-03: Path outside AllowedRoot → ok=false, code=POLICY_DENIED
// INV-GREP-04: grep not in AllowedTools → ok=false, code=POLICY_DENIED
// INV-GREP-05: Non-existent path → ok=false, code=NOT_FOUND
// INV-GREP-06: Empty pattern → ok=false, code=INVALID_ARGS (rejected before exec)
// INV-GREP-07: Timeout exceeded → ok=false, code=TIMEOUT
// INV-GREP-08: Shell injection attempt in pattern → safe (pattern goes to argv, not shell)
// INV-GREP-09: context_lines > 0 → ContextBefore/ContextAfter populated correctly
// INV-GREP-10: literal=true → pattern treated as fixed string, not regex
