# Phase 2 Addendum: `tar` Tool

## Overview
This document specifies the behavior of the `tar` tool.
`tar` is a **read-only** tool — it only **lists** or **extracts** archive contents, it never creates archives.
Implementation must use a pure Go archive library (`archive/tar` from stdlib) — **no subprocess execution**.

## Tool Specifications

### 1. `tar`
Lists the contents of a tar archive, or extracts a single named entry from it.

**Parameters:**
- `path` (string, required): The absolute path to the `.tar`, `.tar.gz`, `.tgz`, `.tar.bz2`, or `.tar.xz` archive.
- `action` (string, required): One of `"list"` or `"extract"`.
- `entry` (string, optional): Required only when `action = "extract"`. The exact path of the entry inside the archive to extract and return as a string (text files only).

**Policy Enforcement:**
- Must call `policyEngine.Evaluate(ctx, "tar", []string{path}, false)` — `isMutating` is `false`.
- Path must be within `allowed_root`.
- The `extract` action reads and returns the file content in memory — it does **not** write anything to disk.

**Returns (action = "list"):**
```json
{
  "entries": [
    {"name": "dist/osmcp_linux_amd64", "size": 8421376, "mode": "-rwxr-xr-x", "is_dir": false},
    {"name": "dist/", "size": 0, "mode": "drwxr-xr-x", "is_dir": true}
  ],
  "count": 2
}
```

**Returns (action = "extract"):**
```json
{
  "entry": "dist/osmcp_linux_amd64",
  "content": "...",
  "size": 8421376
}
```

**Error cases:**
- `POLICY_DENIED` if path is outside `allowed_root`.
- `NOT_FOUND` if the archive file does not exist.
- `INVALID_ARGS` if `action` is not `"list"` or `"extract"`, or if `entry` is missing when `action = "extract"`.
- `EXEC_FAILED` if the archive is corrupt or the requested `entry` does not exist inside it.
- `EXEC_FAILED` if `extract` is called on a binary file (non-UTF-8 content).

## Invariants

- **INV-TAR-01**: `tar` never writes to disk. The `extract` action returns the content in the envelope — not as a file.
- **INV-TAR-02**: The implementation uses pure Go stdlib `archive/tar` — never `exec.Command("tar", ...)`.
- **INV-TAR-03**: Compressed archives (`.gz`, `.bz2`, `.xz`) are transparently decompressed in memory.
- **INV-TAR-04**: Path traversal entries (e.g. `../../etc/passwd`) inside the archive must be rejected with `EXEC_FAILED`.
- **INV-TAR-05**: `list` output is truncated at `max_matches`; `meta.truncated = true` when truncation occurs.
- **INV-TAR-06**: `extract` content is truncated at `max_output_bytes`; `meta.truncated = true` when truncation occurs.

## JSON-RPC MCP Registration
The tool must be self-contained and implement `RegisterMCP(s *server.MCPServer)`.
