# Phase 2 Addendum: `awk` Tool

## Overview
This document specifies the behavior of the `awk` text-processing tool.
`awk` is a **read-only** tool — it processes input and returns transformed text without modifying any file on disk.
Implementation must use a pure Go AWK interpreter (recommended: `github.com/benhoyt/goawk`) — **no subprocess execution**.

## Tool Specifications

### 1. `awk`
Runs an AWK program against the lines of a file and returns the result.

**Parameters:**
- `program` (string, required): The AWK program to execute (e.g. `'{print $2}'`).
- `path` (string, required): The absolute path to the input file.
- `field_separator` (string, optional, default: `" "`): The field separator character (equivalent to `awk -F`).

**Policy Enforcement:**
- Must call `policyEngine.Evaluate(ctx, "awk", []string{path}, false)` — `isMutating` is `false`.
- Path must be within `allowed_root`.

**Returns:**
```json
{
  "output": "column1\ncolumn2\n",
  "lines_processed": 42
}
```

**Error cases:**
- `POLICY_DENIED` if path is outside `allowed_root`.
- `NOT_FOUND` if the file does not exist.
- `INVALID_ARGS` if `program` is empty, or `path` is empty or is a directory.
- `EXEC_FAILED` if the AWK program has a syntax error or runtime error.

## Invariants

- **INV-AWK-01**: `awk` never writes to disk. The original file is always unchanged.
- **INV-AWK-02**: The implementation uses pure Go — never `exec.Command("awk", ...)`.
- **INV-AWK-03**: Output is truncated at `max_output_bytes`; `meta.truncated = true` when truncation occurs.
- **INV-AWK-04**: AWK programs that attempt file system writes (e.g. `print > "/etc/passwd"`) must be rejected with `EXEC_FAILED`.

## JSON-RPC MCP Registration
The tool must be self-contained and implement `RegisterMCP(s *server.MCPServer)`.
