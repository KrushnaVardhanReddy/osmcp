// osmcp — Tar Tool Contract (Phase 2 Addendum)
// Spec reference: docs/specs/phase-2/06_tar_spec.md
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be approved first.

package phase2

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

// TarAction is the operation to perform on the archive.
type TarAction string

const (
	// TarActionList lists the contents of the archive.
	TarActionList TarAction = "list"

	// TarActionExtract extracts and returns a single named entry as a string.
	TarActionExtract TarAction = "extract"
)

// TarArgs defines the input parameters for the tar tool.
// Spec: docs/specs/phase-2/06_tar_spec.md
type TarArgs struct {
	// Path is the absolute path to the archive file (.tar, .tar.gz, .tgz, .tar.bz2, .tar.xz).
	Path string `json:"path"`

	// Action is the operation to perform. One of "list" or "extract".
	Action TarAction `json:"action"`

	// Entry is the exact path of the archive entry to extract.
	// Required when Action == TarActionExtract.
	Entry string `json:"entry,omitempty"`
}

// TarEntry describes a single file or directory inside an archive.
// Used in TarListData.
type TarEntry struct {
	// Name is the path of the entry inside the archive.
	Name string `json:"name"`

	// Size is the uncompressed size in bytes.
	Size int64 `json:"size"`

	// Mode is the human-readable file mode string (e.g. "-rwxr-xr-x").
	Mode string `json:"mode"`

	// IsDir is true if the entry is a directory.
	IsDir bool `json:"is_dir"`
}

// TarListData is the payload for action="list".
type TarListData struct {
	// Entries is the list of files/directories found in the archive.
	Entries []TarEntry `json:"entries"`

	// Count is the total number of entries returned (may be less than total if truncated).
	Count int `json:"count"`
}

// TarExtractData is the payload for action="extract".
type TarExtractData struct {
	// Entry is the path of the entry that was extracted.
	Entry string `json:"entry"`

	// Content is the UTF-8 text content of the extracted entry.
	Content string `json:"content"`

	// Size is the uncompressed size of the entry in bytes.
	Size int64 `json:"size"`
}

// TarTool is the interface the tar tool implementation must satisfy.
// Spec: docs/specs/phase-2/06_tar_spec.md
type TarTool interface {
	// Tar executes the archive operation and returns a uniform response envelope.
	// Envelope.Data is TarListData for action="list" or TarExtractData for action="extract".
	Tar(req TarArgs) contracts.Envelope
}

// Invariants (verified by tests)
//
// INV-TAR-01: tar never writes to disk. IsMutating() must return false.
// INV-TAR-02: Implementation uses pure Go stdlib archive/tar — never exec.Command("tar").
// INV-TAR-03: Compressed archives are transparently decompressed in memory.
// INV-TAR-04: Entries with path traversal patterns (../ or leading /) return ErrExecFailed.
// INV-TAR-05: list output is truncated at max_matches; meta.Truncated=true when truncated.
// INV-TAR-06: extract content is truncated at max_output_bytes; meta.Truncated=true when truncated.
// INV-TAR-07: extract on a non-UTF-8 binary entry returns ErrExecFailed.
