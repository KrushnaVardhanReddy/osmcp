# Task 4: File Inspection Tools (ls, cat, stat, wc)

## Objective
Implement four foundational read-only filesystem tools (`ls`, `cat`, `stat`, `wc`) using Pure Go standard libraries. 

> ⚠️ **Spec-first rule**: Do NOT write any implementation code before reading the specs
> and contracts listed in Prerequisites. 

## Prerequisites
- Task 1 must be fully complete and merged.
- Read `docs/specs/phase-1/02_file_inspection_spec.md` fully.
- Read `docs/contracts/phase-1/file_inspection_contract.go`. Do not alter the structs.

## Implementation Steps

### 1. Implement `ls`
File: `internal/tools/ls.go`
- Implement `Tool` interface (`Name() string` returns `"ls"`).
- Use `filepath.WalkDir` if recursive, or `os.ReadDir` if not recursive.
- Enforce `max_depth` (count separators or keep depth map) and `show_hidden` filters.
- Stop walking and return early if matches reach `policy.Limits().MaxMatches`. Set `meta.Truncated = true`.

### 2. Implement `cat`
File: `internal/tools/cat.go`
- Implement `Tool` interface (`Name() string` returns `"cat"`).
- Reject if path is a directory (`os.Stat`).
- Open file and read 512 bytes to test `http.DetectContentType`. If not text, return `EXEC_FAILED`.
- Read file line by line using `bufio.Scanner` (seek to `start_line`).
- Stop reading if `policy.Limits().MaxOutputBytes` is exceeded. Set `meta.Truncated = true`.

### 3. Implement `stat` and `wc`
File: `internal/tools/stat.go`
- Use `os.Stat` and format into `StatData`.

File: `internal/tools/wc.go`
- Open file and use `bufio.Scanner` to count bytes, words, and lines. 
- Must stream effectively so it doesn't load a massive file entirely into memory.

### 4. Register Tools
File: `cmd/osmcp/main.go`
- Update main to construct and register the four new tools in the `ToolRegistry`.

### 5. Unit Tests
Files: `internal/tools/ls_test.go`, `internal/tools/cat_test.go`, etc.
- Must cover the 8 invariants listed in `file_inspection_contract.go`.
- No mocks. Use temporary directories and real files via `t.TempDir()`.

## Definition of Done
- `make test` passes clean.
- Calling `ls` over MCP returns a structured JSON directory listing.
- Calling `cat` over MCP streams the file, respects start/end lines, and rejects binary files.
- `wc` correctly counts words on a 1GB file without crashing memory (streaming).

## Files to Create
```
internal/tools/ls.go
internal/tools/ls_test.go
internal/tools/cat.go
internal/tools/cat_test.go
internal/tools/stat.go
internal/tools/stat_test.go
internal/tools/wc.go
internal/tools/wc_test.go
```
