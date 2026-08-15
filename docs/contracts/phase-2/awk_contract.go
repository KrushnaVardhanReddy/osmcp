// osmcp — Awk Tool Contract (Phase 2 Addendum)
// Spec reference: docs/specs/phase-2/05_awk_spec.md
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be approved first.

package phase2

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

// AwkArgs defines the input parameters for the awk tool.
// Spec: docs/specs/phase-2/05_awk_spec.md
type AwkArgs struct {
	// Program is the AWK program to execute (e.g. '{print $2}').
	Program string `json:"program"`

	// Path is the absolute path to the input file.
	Path string `json:"path"`

	// FieldSeparator is the AWK field separator (equivalent to awk -F). Default: " ".
	FieldSeparator string `json:"field_separator"`
}

// AwkData is the tool-specific payload returned in Envelope.Data on success.
type AwkData struct {
	// Output is the complete stdout produced by the AWK program.
	Output string `json:"output"`

	// LinesProcessed is the number of input lines the AWK program consumed.
	LinesProcessed int `json:"lines_processed"`
}

// AwkTool is the interface the awk tool implementation must satisfy.
// Spec: docs/specs/phase-2/05_awk_spec.md
type AwkTool interface {
	// Awk executes the AWK program against the file and returns a uniform response envelope.
	Awk(req AwkArgs) contracts.Envelope
}

// Invariants (verified by tests)
//
// INV-AWK-01: awk never writes to disk. IsMutating() must return false.
// INV-AWK-02: Implementation uses pure Go (github.com/benhoyt/goawk) — never exec.Command("awk").
// INV-AWK-03: Output is truncated at max_output_bytes; meta.Truncated=true when truncated.
// INV-AWK-04: AWK programs containing file write redirects (>) must return ErrExecFailed.
