# Task 7: Data Transformation Tools (jq, sed, diff)

## Objective
Implement three in-process data transformation tools (`jq`, `sed`, `diff`) using Pure Go libraries. These tools operate on input strings — they do NOT read from disk, so no path policy checks apply.

> ⚠️ **Spec-first rule**: Read prerequisites before writing any code.

## Prerequisites
- Tasks 1 and 2 must be fully complete and merged.
- Read `docs/specs/phase-1/05_transform_tools_spec.md` fully.
- Read `docs/contracts/phase-1/transform_contract.go`. Do not alter the structs.
- Add the following libraries via `go get`:
  - `github.com/itchyny/gojq` (jq)
  - `github.com/sergi/go-diff/diffmatchpatch` (diff)
  - (sed uses stdlib `regexp` only)

## Implementation Steps

### 1. Implement `jq`
File: `internal/tools/jq.go`
- Implement `Tool` interface (`Name()` returns `"jq"`).
- Parse `args.Filter` using `gojq.Parse()`. On error → `INVALID_ARGS`.
- Unmarshal `args.Input` as JSON. On error → `INVALID_ARGS`.
- Compile and run the query with `gojq.Compile(query).Run(input)`.
- Collect all output values. Re-marshal as compact or pretty JSON based on `args.Compact`.
- Set `OutputType` by inspecting the type of the first result.

### 2. Implement `sed`
File: `internal/tools/sed.go`
- Implement `Tool` interface (`Name()` returns `"sed"`).
- Parse `args.Expression` (must match `s/pattern/replacement/flags`). On format error → `INVALID_ARGS`.
- Compile the pattern with `regexp.Compile` (or `regexp.CompilePOSIX` if `i` flag is needed). On error → `INVALID_ARGS`.
- Apply replacement:
  - If `g` flag: `re.ReplaceAllString(input, replacement)`.
  - If no `g` flag: `re.ReplaceAllLiteralString` on the first match only.
- Count `replacements_made` by comparing match count before and after.
- Enforce `MaxOutputBytes`. If result exceeds limit, truncate and set `meta.Truncated = true`.

### 3. Implement `diff`
File: `internal/tools/diff.go`
- Implement `Tool` interface (`Name()` returns `"diff"`).
- Use `diffmatchpatch.New().DiffMain(a, b, true)`.
- Convert diffs to a unified patch string using `dmp.DiffToPretty()` or custom unified format with `context_lines`.
- Count `additions` (diffs with `DiffInsert`) and `deletions` (diffs with `DiffDelete`).
- Set `identical = true` if the patch is empty.
- Enforce `MaxOutputBytes` on the patch string. Truncate and set `meta.Truncated = true` if exceeded.

### 4. Register Tools
File: `cmd/osmcp/main.go`
- Construct and register `JqTool`, `SedTool`, `DiffTool` in the `ToolRegistry`.

### 5. Unit Tests
Files: `internal/tools/jq_test.go`, `internal/tools/sed_test.go`, `internal/tools/diff_test.go`
- Must cover all 9 invariants in `transform_contract.go`.
- No mocks. No file I/O needed.

## Definition of Done
- `go test -race ./...` passes.
- `jq` correctly filters `.users[].name` from a sample JSON object.
- `sed` correctly applies `s/foo/bar/g` globally and `s/foo/bar/` to only the first match.
- `diff` returns an empty patch and `identical: true` for matching strings.

## Files to Create
```
internal/tools/jq.go
internal/tools/jq_test.go
internal/tools/sed.go
internal/tools/sed_test.go
internal/tools/diff.go
internal/tools/diff_test.go
```
