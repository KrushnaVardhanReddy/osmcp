# Spec: Git Intelligence Tools (git_status, git_diff, git_log)
# Phase 1 — Read-only Git operations

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Provide a safe, capability-bound way for agents to inspect the state of a local Git repository. These tools must be implemented in **Pure Go** using the `github.com/go-git/go-git/v5` library. `os/exec` calling the `git` binary is strictly forbidden, ensuring zero shell-injection risk and no host-tool dependency.

---

## 2. Tool: `git_status` (Working Tree Status)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the repository root (or any subdirectory within it)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "branch": "feature/dev",
  "head_commit": "a1b2c3d",
  "clean": false,
  "staged": [
    { "path": "internal/policy/engine.go", "code": "M" }
  ],
  "unstaged": [
    { "path": "go.mod", "code": "M" }
  ],
  "untracked": [
    "tmp/scratch.go"
  ]
}
```

Status codes mirror the standard single-character git status codes:
- `A` — Added
- `M` — Modified
- `D` — Deleted
- `R` — Renamed
- `?` — Untracked

---

## 3. Tool: `git_diff` (File or Commit Diff)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the repository root."
    },
    "file": {
      "type": "string",
      "description": "Optional. Relative file path to restrict the diff to a single file."
    },
    "from_commit": {
      "type": "string",
      "description": "Optional. Starting commit hash (defaults to HEAD~1)."
    },
    "to_commit": {
      "type": "string",
      "description": "Optional. Ending commit hash (defaults to HEAD)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "patches": [
    {
      "file": "internal/policy/engine.go",
      "additions": 12,
      "deletions": 3,
      "diff": "@@ -10,6 +10,8 @@ func Evaluate(...) {\n..."
    }
  ],
  "total_additions": 12,
  "total_deletions": 3
}
```

### Truncation
If the combined diff output exceeds `policy.Limits().MaxOutputBytes`, stop adding patches, set `meta.truncated = true`, and return the patches collected so far.

---

## 4. Tool: `git_log` (Commit History)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the repository root."
    },
    "max_commits": {
      "type": "integer",
      "default": 20,
      "minimum": 1,
      "maximum": 200,
      "description": "Maximum number of commits to return."
    },
    "file": {
      "type": "string",
      "description": "Optional. If set, only return commits that touched this file path."
    },
    "branch": {
      "type": "string",
      "description": "Optional. Branch to read history from (defaults to current HEAD)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "commits": [
    {
      "hash": "a1b2c3d4e5f6...",
      "short_hash": "a1b2c3d",
      "author": "Krushna VardhanReddy",
      "email": "krushna@example.com",
      "date": "2026-08-14T10:00:00Z",
      "message": "docs: add file inspection spec"
    }
  ],
  "count": 1
}
```

### Truncation
Stop iterating commits once `min(args.max_commits, policy.Limits().MaxMatches)` is reached. Set `meta.truncated = true` if the log was cut short.

---

## 5. Execution & Policy Rules

1. **Pure Go:** All three tools use `github.com/go-git/go-git/v5` exclusively. `os/exec` and the `git` binary must never be called.
2. **Policy Evaluation:** `PolicyEngine.Evaluate(toolName, []string{args.Path}, false)` must be called before opening the repository.
3. **Symlinks:** Resolve symlinks on `args.Path` before policy check (`filepath.EvalSymlinks`).
4. **Repository Discovery:** Use `git.PlainOpenWithOptions` with `DetectDotGit: true` so that calling with any subdirectory path still works correctly.
5. **Read-Only:** None of these tools can create commits, push, or modify any git state. `go-git` handles this naturally by opening repos in read-only mode.

---

## 6. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| `path` outside `allowed_root` | `ok: false`, `code: POLICY_DENIED` |
| `path` is not inside a git repo | `ok: false`, `code: EXEC_FAILED` ("Not a git repository") |
| Repo is clean | `git_status` returns `clean: true`, empty arrays |
| `from_commit` hash not found | `ok: false`, `code: INVALID_ARGS` |
| `git_diff` output exceeds `max_output_bytes` | `ok: true`, `meta.truncated: true` |
| `git_log` `max_commits` > policy `max_matches` | Use `min()` of the two, set `meta.truncated: true` |
