# Task 20: Built-in Filter Parameters (Phase 2 Addendum)

## Goal
Enhance existing Phase 1 tools with built-in server-side filter parameters so agents
can avoid unnecessary large intermediate responses. This eliminates the most common
"ls | grep" and "git log | grep" patterns by keeping filtering inside a single tool call.

## Spec Reference
`docs/specs/phase-1/02_file_inspection_spec.md` — ls addendum  
`docs/specs/phase-1/03_git_tools_spec.md` — git_log addendum

## Contract Reference
`docs/contracts/phase-1/file_inspection_contract.go` — LsArgs addendum  
`docs/contracts/phase-1/git_contract.go` — GitLogArgs addendum

## Changes Required

### 1. `ls` — add `pattern` parameter

**New optional field in `LsArgs`:**
```go
// Pattern is an optional glob pattern to filter entries by name (e.g. "*.go").
// If empty, all entries are returned (existing behaviour — fully backwards compatible).
Pattern string `json:"pattern,omitempty"`
```

**Behaviour:**
- Use `path.Match(pattern, entry.Name)` from Go stdlib to filter `LsEntry` results.
- If `Pattern` is empty or omitted, behave exactly as before (return all entries).
- Apply filtering BEFORE the `max_matches` truncation check — i.e., count only matched entries toward the limit.
- Invalid glob patterns (e.g. unmatched `[`) return `INVALID_ARGS`.

**Example call:**
```json
{ "path": "/src", "pattern": "*.go" }
```

### 2. `git_log` — add `author`, `since`, `until` parameters

**New optional fields in `GitLogArgs` (or equivalent struct):**
```go
// Author filters commits to only those whose author name or email contains this string.
// Case-insensitive substring match. If empty, no author filtering is applied.
Author string `json:"author,omitempty"`

// Since filters commits to those after this date (RFC3339, e.g. "2024-01-01T00:00:00Z").
// If empty, no lower date bound is applied.
Since string `json:"since,omitempty"`

// Until filters commits to those before this date (RFC3339).
// If empty, no upper date bound is applied.
Until string `json:"until,omitempty"`
```

**Behaviour:**
- Use `go-git/v5` `LogOptions` fields `Since` and `Until` (they accept `*time.Time`).
- For `Author`, filter in Go after fetching log entries: `strings.Contains(strings.ToLower(commit.Author.Name), strings.ToLower(author))`.
- All three parameters are optional and fully backwards-compatible. If all are empty, git_log behaves exactly as before.
- Invalid RFC3339 date strings return `INVALID_ARGS`.

## Implementation Details

1. **No breaking changes** — all new parameters are optional with `omitempty`. Existing callers are unaffected.
2. **No new files** — modify the existing implementation files:
   - `internal/tools/ls.go` (or wherever ls is implemented)
   - `internal/tools/git_tools.go` (or wherever git_log is implemented)
3. **No contract file modifications** — add the new fields directly in the implementation structs. The contract files in `docs/contracts/` are read-only documentation.

## Testing
- `ls` with `pattern: "*.go"` returns only `.go` files from a mixed directory.
- `ls` with no `pattern` returns all files (backwards compatibility).
- `ls` with an invalid glob `pattern: "["` returns `INVALID_ARGS`.
- `git_log` with `author: "alice"` returns only commits from Alice.
- `git_log` with `since: "2024-01-01T00:00:00Z"` returns only commits after that date.
- `git_log` with `since` AND `until` returns commits in the date window.
- `git_log` with no filter parameters returns all commits (backwards compatibility).
- `git_log` with `since: "not-a-date"` returns `INVALID_ARGS`.

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
- Any other tool implementations
