// osmcp — Data Transformation Tools Contract (Phase 1)
//
// Spec reference: docs/specs/phase-1/05_transform_tools_spec.md
// Libraries:
//   - jq:   github.com/itchyny/gojq
//   - sed:  regexp (stdlib)
//   - diff: github.com/sergi/go-diff/diffmatchpatch

package contracts

// ─────────────────────────────────────────────────────────────────────────────
// jq (JSON Query)
// ─────────────────────────────────────────────────────────────────────────────

type JqArgs struct {
	Input   string `json:"input" jsonschema:"required,description=JSON string to query."`
	Filter  string `json:"filter" jsonschema:"required,description=jq filter expression."`
	Compact bool   `json:"compact" jsonschema:"default=false,description=If true output is compact."`
}

type JqData struct {
	Result     string `json:"result"`
	OutputType string `json:"output_type"` // "object", "array", "string", "number", "boolean", "null"
}

// ─────────────────────────────────────────────────────────────────────────────
// sed (Stream Edit)
// ─────────────────────────────────────────────────────────────────────────────

type SedArgs struct {
	Input      string `json:"input" jsonschema:"required,description=Text content to transform."`
	Expression string `json:"expression" jsonschema:"required,description=sed-style substitute expression: s/pattern/replacement/flags."`
}

type SedData struct {
	Result           string `json:"result"`
	ReplacementsMade int    `json:"replacements_made"`
}

// ─────────────────────────────────────────────────────────────────────────────
// diff (Text Comparison)
// ─────────────────────────────────────────────────────────────────────────────

type DiffArgs struct {
	A            string `json:"a" jsonschema:"required,description=Original text (left side)."`
	B            string `json:"b" jsonschema:"required,description=Modified text (right side)."`
	ContextLines int    `json:"context_lines" jsonschema:"default=3,minimum=0,maximum=10"`
}

type DiffData struct {
	Patch     string `json:"patch"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Identical bool   `json:"identical"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariants
// ─────────────────────────────────────────────────────────────────────────────

// INV-TRANSFORM-01: Pure Go — os/exec is never called for any transform tool.
// INV-TRANSFORM-02: No file I/O — these tools operate on input strings only; allowed_root is irrelevant.
// INV-TRANSFORM-03: jq invalid JSON input → ok=false, INVALID_ARGS.
// INV-TRANSFORM-04: jq invalid filter syntax → ok=false, INVALID_ARGS.
// INV-TRANSFORM-05: sed invalid expression format → ok=false, INVALID_ARGS.
// INV-TRANSFORM-06: sed invalid regex pattern → ok=false, INVALID_ARGS.
// INV-TRANSFORM-07: diff on identical strings → ok=true, identical=true, patch="".
// INV-TRANSFORM-08: Output exceeds MaxOutputBytes → ok=true, meta.Truncated=true.
// INV-TRANSFORM-09: Input exceeds MaxOutputBytes → ok=false, INVALID_ARGS.
