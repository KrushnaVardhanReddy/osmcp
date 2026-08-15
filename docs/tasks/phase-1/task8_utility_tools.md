# Task 8: Utility Tools (tree, head, tail, du)

## Objective
Implement four high-value utility tools (`tree`, `head`, `tail`, `du`) that make `osmcp` significantly more useful for AI coding agents compared to competing MCP filesystem servers.

> ⚠️ **Spec-first rule**: Read prerequisites before writing any code.

## Prerequisites
- Tasks 1 and 2 must be fully complete and merged.
- Read `docs/specs/phase-1/06_utility_tools_spec.md` fully.
- Read `docs/contracts/phase-1/utility_contract.go`. Do not alter the structs.

## Implementation Steps

### 1. Implement `tree`
File: `internal/tools/tree.go`
- Implement `Tool` interface (`Name()` returns `"tree"`).
- Walk with `filepath.WalkDir`. Track depth and maintain a prefix stack for box-drawing characters.
- Use `├── ` for non-last entries and `└── ` for the last entry in a directory. Indent children with `│   ` or `    `.
- Apply `show_hidden` and `dirs_only` filters.
- Count `dirs` and `files` separately. Stop at `MaxMatches`, append `... (truncated)`, set `meta.Truncated = true`.

### 2. Implement `head`
File: `internal/tools/head.go`
- Implement `Tool` interface (`Name()` returns `"head"`).
- Reject directories and binary files (same binary check as `cat`).
- Use `bufio.Scanner` to read exactly `args.Lines` lines.
- Set `eof_reached = true` if the scanner hits EOF before reaching the line limit.

### 3. Implement `tail`
File: `internal/tools/tail.go`
- Implement `Tool` interface (`Name()` returns `"tail"`).
- Reject directories and binary files.
- **Must use a ring buffer** of capacity `args.Lines` — do NOT read the full file into memory.
  Implementation pattern:
  ```go
  buf := make([]string, args.Lines)
  idx := 0
  count := 0
  // scan all lines:
  buf[idx % args.Lines] = line
  idx++; count++
  // After scan, reconstruct from buf[(idx)%N] onwards
  ```

### 4. Implement `du`
File: `internal/tools/du.go`
- Implement `Tool` interface (`Name()` returns `"du"`).
- Walk with `filepath.WalkDir`. Accumulate `Size()` for each file (not dirs).
- Track size per subdirectory at depth <= `args.MaxDepth`.
- Format bytes to human-readable with 1024 divisors: B, KB, MB, GB.

### 5. Register Tools
File: `cmd/osmcp/main.go`
- Construct and register `TreeTool`, `HeadTool`, `TailTool`, `DuTool` in the `ToolRegistry`.

### 6. Unit Tests
Files: `internal/tools/tree_test.go`, `internal/tools/head_test.go`, `internal/tools/tail_test.go`, `internal/tools/du_test.go`
- Use `t.TempDir()` for all file/dir creation.
- Must cover all 9 invariants in `utility_contract.go`.
- Test `tail` on a file with 10,000 lines requesting last 20 — confirm only last 20 are returned.

## Definition of Done
- `go test -race ./...` passes.
- `tree` output on the osmcp repo itself shows the correct box-drawing structure.
- `tail` on a 100MB file does not cause memory spike (stays O(N lines) in memory).
- `du` shows per-subdirectory breakdown at `max_depth: 2`.

## Files to Create
```
internal/tools/tree.go
internal/tools/tree_test.go
internal/tools/head.go
internal/tools/head_test.go
internal/tools/tail.go
internal/tools/tail_test.go
internal/tools/du.go
internal/tools/du_test.go
```
