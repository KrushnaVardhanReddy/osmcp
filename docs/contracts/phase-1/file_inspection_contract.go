// osmcp — File Inspection Tools Contract (Phase 1)
//
// Spec reference: docs/specs/phase-1/02_file_inspection_spec.md
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be approved first.

package contracts

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// ls (List Directory)
// ─────────────────────────────────────────────────────────────────────────────

type LsArgs struct {
	Path       string `json:"path" jsonschema:"required,description=Absolute path to the directory."`
	Recursive  bool   `json:"recursive" jsonschema:"default=false,description=If true, walks subdirectories."`
	MaxDepth   int    `json:"max_depth" jsonschema:"default=1,minimum=1,maximum=10,description=Max depth for recursion."`
	ShowHidden bool   `json:"show_hidden" jsonschema:"default=false,description=Show files starting with dot."`
}

type LsData struct {
	Entries []LsEntry `json:"entries"`
	Count   int       `json:"count"`
}

type LsEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

// ─────────────────────────────────────────────────────────────────────────────
// cat (Read File)
// ─────────────────────────────────────────────────────────────────────────────

type CatArgs struct {
	Path      string `json:"path" jsonschema:"required,description=Absolute path to file."`
	StartLine int    `json:"start_line" jsonschema:"default=1,minimum=1,description=Line number to start reading from."`
	EndLine   *int   `json:"end_line,omitempty" jsonschema:"description=Line to stop reading at (inclusive)."`
}

type CatData struct {
	Content       string `json:"content"`
	LinesReturned int    `json:"lines_returned"`
	EOFReached    bool   `json:"eof_reached"`
}

// ─────────────────────────────────────────────────────────────────────────────
// stat (File Metadata)
// ─────────────────────────────────────────────────────────────────────────────

type StatArgs struct {
	Path string `json:"path" jsonschema:"required,description=Absolute path to file or directory."`
}

type StatData struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

// ─────────────────────────────────────────────────────────────────────────────
// wc (Word/Line Count)
// ─────────────────────────────────────────────────────────────────────────────

type WcArgs struct {
	Path string `json:"path" jsonschema:"required,description=Absolute path to file."`
}

type WcData struct {
	Lines int `json:"lines"`
	Words int `json:"words"`
	Bytes int `json:"bytes"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariants (verified by tests)
// ─────────────────────────────────────────────────────────────────────────────

// INV-FILE-01: Pure Go — os/exec is never used for these four tools.
// INV-FILE-02: Policy Check — Path outside AllowedRoot returns ok=false, POLICY_DENIED.
// INV-FILE-03: Missing File — Missing path returns ok=false, NOT_FOUND.
// INV-FILE-04: cat Binary — Calling cat on a binary file returns ok=false, EXEC_FAILED.
// INV-FILE-05: ls Truncation — Walking stops if max_matches is hit (meta.Truncated=true).
// INV-FILE-06: cat Truncation — Reading stops if max_output_bytes is hit (meta.Truncated=true, eof_reached=false).
// INV-FILE-07: cat on Dir — Calling cat on a directory returns ok=false, INVALID_ARGS.
// INV-FILE-08: ls on File — Calling ls on a file returns ok=true, 1 entry for the file itself.
