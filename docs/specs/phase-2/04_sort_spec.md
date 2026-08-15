# Phase 2 Addendum: `sort` Tool

## Overview
This document specifies the behavior of the `sort` text-processing tool.
`sort` is a **read-only** tool — it reads input and returns sorted output without modifying any file on disk.

## Tool Specifications

### 1. `sort`
Reads the contents of a file and returns its lines in sorted order.

**Parameters:**
- `path` (string, required): The absolute path to the file to sort.
- `reverse` (boolean, optional, default: `false`): If true, sort in descending order.
- `unique` (boolean, optional, default: `false`): If true, deduplicate adjacent identical lines (equivalent to `sort -u`).
- `numeric` (boolean, optional, default: `false`): If true, sort by numeric value rather than lexicographic order (equivalent to `sort -n`).

**Policy Enforcement:**
- Must call `policyEngine.Evaluate(ctx, "sort", []string{path}, false)` — `isMutating` is `false`.
- Path must be within `allowed_root`.

**Returns:**
```json
{
  "lines": ["bar", "baz", "foo"],
  "count": 3
}
```

**Error cases:**
- `POLICY_DENIED` if path is outside `allowed_root`.
- `NOT_FOUND` if the file does not exist.
- `INVALID_ARGS` if `path` is empty or points to a directory.

## Invariants

- **INV-SORT-01**: `sort` never writes to disk. The original file is always unchanged.
- **INV-SORT-02**: Output is truncated at `max_output_bytes`; `meta.truncated = true` when truncation occurs.
- **INV-SORT-03**: An empty file returns `{"lines": [], "count": 0}`, not an error.
- **INV-SORT-04**: `unique` deduplication applies after sorting (identical to POSIX `sort -u`).

## JSON-RPC MCP Registration
The tool must be self-contained and implement `RegisterMCP(s *server.MCPServer)`.
