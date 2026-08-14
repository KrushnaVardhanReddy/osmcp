# Task 5: Git Intelligence Tools (git_status, git_diff, git_log)

## Objective
Implement three read-only Git inspection tools (`git_status`, `git_diff`, `git_log`) using the Pure Go `github.com/go-git/go-git/v5` library.

> ⚠️ **Spec-first rule**: Do NOT write any implementation code before reading the specs
> and contracts listed in Prerequisites.

## Prerequisites
- Tasks 1 and 2 must be fully complete and merged.
- Read `docs/specs/phase-1/03_git_tools_spec.md` fully.
- Read `docs/contracts/phase-1/git_contract.go`. Do not alter the structs.
- Add `github.com/go-git/go-git/v5` to `go.mod` via `go get`.

## Implementation Steps

### 1. Implement `git_status`
File: `internal/tools/git_status.go`
- Implement `Tool` interface (`Name()` returns `"git_status"`).
- Open the repo using `git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})`.
- Call `repo.Head()` to get the branch name and HEAD commit short hash.
- Call `worktree.Status()` to get the full file status map.
- Walk the status map and sort entries into `staged`, `unstaged`, and `untracked` arrays.
- Return `GitStatusData`. If no files are dirty, set `clean: true`.

### 2. Implement `git_diff`
File: `internal/tools/git_diff.go`
- Implement `Tool` interface (`Name()` returns `"git_diff"`).
- Resolve `from_commit` and `to_commit` hashes. Default: HEAD~1 to HEAD.
- If a commit hash is not found, return `INVALID_ARGS`.
- Use `fromCommit.Tree()` and `toCommit.Tree()` to get the two trees.
- Call `fromTree.Diff(toTree)` to get the patch list.
- If `file` arg is set, filter patches to only include the matching file.
- Track total bytes. If output exceeds `policy.Limits().MaxOutputBytes`, stop and set `meta.Truncated = true`.

### 3. Implement `git_log`
File: `internal/tools/git_log.go`
- Implement `Tool` interface (`Name()` returns `"git_log"`).
- Open the repo and call `repo.Log(&git.LogOptions{})`.
- If `branch` is set, resolve it to a hash first and pass via `git.LogOptions{From: hash}`.
- If `file` is set, use `git.LogOptions{FileName: &args.File}` to filter.
- Iterate commits up to `min(args.MaxCommits, policy.Limits().MaxMatches)`.
- Return `GitLogData`. If the iterator had more commits, set `meta.Truncated = true`.

### 4. Register Tools
File: `cmd/osmcp/main.go`
- Construct and register `git_status`, `git_diff`, and `git_log` in the `ToolRegistry`.

### 5. Unit Tests
Files: `internal/tools/git_status_test.go`, `internal/tools/git_diff_test.go`, `internal/tools/git_log_test.go`
- Use `go-git` in-memory repositories (`memory.NewStorage()`) for fast, self-contained tests.
- Must cover all 8 invariants from `git_contract.go`.
- No mocks. Use real in-memory git repos with programmatically created commits.

## Definition of Done
- `make test` passes clean with `go test -race ./...`.
- `git_status` correctly identifies staged, unstaged, and untracked files.
- `git_diff` correctly returns unified-diff patches between two commits.
- `git_log` correctly filters by file path when `file` argument is provided.
- All three tools return `POLICY_DENIED` for paths outside `allowed_root`.

## Files to Create
```
internal/tools/git_status.go
internal/tools/git_status_test.go
internal/tools/git_diff.go
internal/tools/git_diff_test.go
internal/tools/git_log.go
internal/tools/git_log_test.go
```
