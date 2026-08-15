// osmcp — find Tool Contract (Phase 1)
//
// Spec reference: docs/specs/phase-1/04_find_tool_spec.md
// Library: path/filepath (stdlib only — no os/exec)

package contracts

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// find
// ─────────────────────────────────────────────────────────────────────────────

type FindArgs struct {
	Path          string     `json:"path" jsonschema:"required,description=Absolute path to search root."`
	Name          *string    `json:"name,omitempty" jsonschema:"description=Glob pattern to match file names."`
	Type          string     `json:"type" jsonschema:"enum=file|dir|any,default=any"`
	MinSize       *int64     `json:"min_size,omitempty" jsonschema:"description=Minimum file size in bytes."`
	MaxSize       *int64     `json:"max_size,omitempty" jsonschema:"description=Maximum file size in bytes."`
	ModifiedAfter *time.Time `json:"modified_after,omitempty" jsonschema:"description=Only return files modified after this ISO 8601 timestamp."`
	MaxDepth      int        `json:"max_depth" jsonschema:"default=10,minimum=1,maximum=20"`
}

type FindData struct {
	Matches []FindEntry `json:"matches"`
	Count   int         `json:"count"`
}

type FindEntry struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariants
// ─────────────────────────────────────────────────────────────────────────────

// INV-FIND-01: Pure Go — filepath.WalkDir only, no os/exec.
// INV-FIND-02: Policy Check — path outside AllowedRoot returns ok=false, POLICY_DENIED.
// INV-FIND-03: Missing Dir — path does not exist returns ok=false, NOT_FOUND.
// INV-FIND-04: No Matches — no files match filters returns ok=true, empty matches, count=0.
// INV-FIND-05: Truncation — walk stops at MaxMatches, meta.Truncated=true.
// INV-FIND-06: Glob Error — invalid name glob returns ok=false, INVALID_ARGS.
// INV-FIND-07: No Symlink Follow — symlinks in the walk are reported but not followed.
