# Spec: find Tool
# Phase 1 — Read-only filesystem discovery

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Provide a safe, capability-bound way for agents to discover files by name pattern, type, size, or modification time. Implemented in **Pure Go** using `path/filepath.WalkDir`. `os/exec` calling the `find` binary is strictly forbidden.

---

## 2. Tool: `find`

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the directory to search."
    },
    "name": {
      "type": "string",
      "description": "Optional glob pattern to match file names (e.g. '*.go', 'main.*')."
    },
    "type": {
      "type": "string",
      "enum": ["file", "dir", "any"],
      "default": "any",
      "description": "Filter by entry type."
    },
    "min_size": {
      "type": "integer",
      "description": "Optional minimum file size in bytes (inclusive)."
    },
    "max_size": {
      "type": "integer",
      "description": "Optional maximum file size in bytes (inclusive)."
    },
    "modified_after": {
      "type": "string",
      "format": "date-time",
      "description": "Optional ISO 8601 timestamp. Only returns files modified after this time."
    },
    "max_depth": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 20,
      "description": "Maximum directory depth to walk."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "matches": [
    {
      "path": "/project/internal/policy/engine.go",
      "name": "engine.go",
      "is_dir": false,
      "size": 2048,
      "mod_time": "2026-08-14T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Truncation
Stop walking once `policy.Limits().MaxMatches` is reached. Set `meta.truncated = true`.

---

## 3. Execution & Policy Rules

1. **Pure Go:** Use `filepath.WalkDir` only. `os/exec` is strictly forbidden.
2. **Policy Evaluation:** Call `PolicyEngine.Evaluate("find", []string{args.Path}, false)` before walking.
3. **Symlinks:** Resolve `args.Path` with `filepath.EvalSymlinks` before the policy check. Do NOT follow symlinks during the walk (`filepath.WalkDir` does not follow them by default — preserve this behaviour).
4. **Glob matching:** Use `filepath.Match(args.Name, entry.Name())` for name patterns.

---

## 4. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| `path` outside `allowed_root` | `ok: false`, `code: POLICY_DENIED` |
| `path` does not exist | `ok: false`, `code: NOT_FOUND` |
| No files match filters | `ok: true`, `data.matches: []`, `count: 0` |
| Walk hits `max_matches` | `ok: true`, `meta.truncated: true` |
| `name` glob is invalid syntax | `ok: false`, `code: INVALID_ARGS` |
