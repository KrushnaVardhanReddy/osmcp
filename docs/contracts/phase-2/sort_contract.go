// osmcp — Sort Tool Contract (Phase 2 Addendum)
// Spec reference: docs/specs/phase-2/04_sort_spec.md
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be approved first.

package phase2

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

// SortArgs defines the input parameters for the sort tool.
// Spec: docs/specs/phase-2/04_sort_spec.md
type SortArgs struct {
	// Path is the absolute path to the file to sort.
	Path string `json:"path"`

	// Reverse sorts in descending order when true. Default: false.
	Reverse bool `json:"reverse"`

	// Unique removes duplicate adjacent lines (sort -u). Default: false.
	Unique bool `json:"unique"`

	// Numeric sorts by numeric value instead of lexicographic order (sort -n). Default: false.
	Numeric bool `json:"numeric"`
}

// SortData is the tool-specific payload returned in Envelope.Data on success.
type SortData struct {
	// Lines is the sorted list of lines from the file.
	Lines []string `json:"lines"`

	// Count is the number of lines returned (after deduplication if unique=true).
	Count int `json:"count"`
}

// SortTool is the interface the sort tool implementation must satisfy.
// Spec: docs/specs/phase-2/04_sort_spec.md
type SortTool interface {
	// Sort executes the sort operation and returns a uniform response envelope.
	Sort(req SortArgs) contracts.Envelope
}

// Invariants (verified by tests)
//
// INV-SORT-01: sort never writes to disk. IsMutating() must return false.
// INV-SORT-02: Output is truncated at max_output_bytes; meta.Truncated=true when truncated.
// INV-SORT-03: An empty file returns SortData{Lines:[]string{}, Count:0}, not an error.
// INV-SORT-04: unique deduplication applies after sorting (identical to POSIX sort -u).
