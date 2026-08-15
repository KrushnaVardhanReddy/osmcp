# Task 6: find Tool

## Objective
Implement the `find` filesystem discovery tool using Pure Go stdlib (`path/filepath`).

> ⚠️ **Spec-first rule**: Read prerequisites before writing any code.

## Prerequisites
- Tasks 1 and 2 must be fully complete and merged.
- Read `docs/specs/phase-1/04_find_tool_spec.md` fully.
- Read `docs/contracts/phase-1/find_contract.go`. Do not alter the structs.

## Implementation Steps

### 1. Implement `find`
File: `internal/tools/find.go`
- Implement `Tool` interface (`Name()` returns `"find"`).
- Validate `args.Name` glob with `filepath.Match` before walking. If invalid, return `INVALID_ARGS`.
- Walk the tree with `filepath.WalkDir`. Track depth by counting `string.Count(path, string(os.PathSeparator))` relative to the root.
- For each entry apply all active filters (name glob, type, min/max size, modified_after) using AND logic.
- Stop walking when matches reach `policy.Limits().MaxMatches`. Set `meta.Truncated = true`.
- Do NOT follow symlinks (this is the default `filepath.WalkDir` behavior — do not change it).

### 2. Register Tool
File: `cmd/osmcp/main.go`
- Construct and register `FindTool` in the `ToolRegistry`.

### 3. Unit Tests
File: `internal/tools/find_test.go`
- Use `t.TempDir()` and `os.MkdirAll` to build a real temp directory tree.
- Must cover all 7 invariants in `find_contract.go`.

## Definition of Done
- `go test -race ./...` passes.
- Calling `find` with `name: "*.go"` returns only `.go` files.
- Calling `find` with `type: "dir"` returns only directories.
- Size and time filters correctly exclude non-matching entries.

## Files to Create
```
internal/tools/find.go
internal/tools/find_test.go
```
