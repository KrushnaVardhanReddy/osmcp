// osmcp — Utility Tools Contract (Phase 1)
//
// Spec reference: docs/specs/phase-1/06_utility_tools_spec.md
// Libraries: stdlib only (os, io, bufio, path/filepath)

package contracts

// ─────────────────────────────────────────────────────────────────────────────
// tree (Directory Tree Visualization)
// ─────────────────────────────────────────────────────────────────────────────

type TreeArgs struct {
	Path       string `json:"path" jsonschema:"required,description=Absolute path to the root directory."`
	MaxDepth   int    `json:"max_depth" jsonschema:"default=3,minimum=1,maximum=10"`
	ShowHidden bool   `json:"show_hidden" jsonschema:"default=false"`
	DirsOnly   bool   `json:"dirs_only" jsonschema:"default=false"`
}

type TreeData struct {
	Tree  string `json:"tree"`  // Box-drawing ASCII tree string
	Dirs  int    `json:"dirs"`
	Files int    `json:"files"`
}

// ─────────────────────────────────────────────────────────────────────────────
// head (First N Lines)
// ─────────────────────────────────────────────────────────────────────────────

type HeadArgs struct {
	Path  string `json:"path" jsonschema:"required"`
	Lines int    `json:"lines" jsonschema:"default=10,minimum=1,maximum=10000"`
}

type HeadData struct {
	Content      string `json:"content"`
	LinesReturned int   `json:"lines_returned"`
	EOFReached   bool   `json:"eof_reached"`
}

// ─────────────────────────────────────────────────────────────────────────────
// tail (Last N Lines)
// ─────────────────────────────────────────────────────────────────────────────

type TailArgs struct {
	Path  string `json:"path" jsonschema:"required"`
	Lines int    `json:"lines" jsonschema:"default=10,minimum=1,maximum=10000"`
}

type TailData struct {
	Content       string `json:"content"`
	LinesReturned int    `json:"lines_returned"`
}

// ─────────────────────────────────────────────────────────────────────────────
// du (Disk Usage)
// ─────────────────────────────────────────────────────────────────────────────

type DuArgs struct {
	Path     string `json:"path" jsonschema:"required"`
	MaxDepth int    `json:"max_depth" jsonschema:"default=1,minimum=1,maximum=5"`
}

type DuData struct {
	TotalBytes int64       `json:"total_bytes"`
	TotalHuman string      `json:"total_human"` // e.g. "512 KB"
	Breakdown  []DuEntry   `json:"breakdown"`
}

type DuEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Human string `json:"human"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariants
// ─────────────────────────────────────────────────────────────────────────────

// INV-UTIL-01: Pure Go — os/exec never called; stdlib only.
// INV-UTIL-02: Policy Check — path outside AllowedRoot returns ok=false, POLICY_DENIED.
// INV-UTIL-03: Missing Path — path does not exist returns ok=false, NOT_FOUND.
// INV-UTIL-04: head/tail on dir — returns ok=false, INVALID_ARGS.
// INV-UTIL-05: head/tail on binary — returns ok=false, EXEC_FAILED.
// INV-UTIL-06: tree on file — returns ok=true, single-entry tree.
// INV-UTIL-07: tree truncation — entries stop at MaxMatches, meta.Truncated=true.
// INV-UTIL-08: tail memory — uses ring buffer; does not load full file into memory.
// INV-UTIL-09: du human format — uses 1024-divisor binary prefix (KB, MB, GB).
