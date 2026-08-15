# Spec: Utility Tools (tree, head, tail, du)
# Phase 1 — Read-only filesystem utilities

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Provide high-value utility tools that are heavily used by AI coding agents. All are implemented in **Pure Go** using stdlib (`os`, `io`, `bufio`, `path/filepath`). `os/exec` is strictly forbidden.

---

## 2. Tool: `tree` (Directory Tree)

Renders a visual directory tree. This is one of the most frequently used tools by coding agents for understanding project structure quickly.

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the directory root."
    },
    "max_depth": {
      "type": "integer",
      "default": 3,
      "minimum": 1,
      "maximum": 10,
      "description": "Maximum depth to display."
    },
    "show_hidden": {
      "type": "boolean",
      "default": false,
      "description": "Include files and dirs starting with a dot."
    },
    "dirs_only": {
      "type": "boolean",
      "default": false,
      "description": "If true, only show directories."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "tree": "osmcp\n├── cmd\n│   └── osmcp\n│       └── main.go\n├── internal\n│   ├── policy\n│   │   └── engine.go\n│   └── tools\n└── go.mod",
  "dirs": 4,
  "files": 3
}
```

The `tree` string uses the standard box-drawing characters: `├──`, `└──`, `│`.

### Truncation
If entries exceed `policy.Limits().MaxMatches`, stop walking and append `... (truncated)` to the tree string. Set `meta.truncated = true`.

---

## 3. Tool: `head` (First N Lines)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the file."
    },
    "lines": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 10000,
      "description": "Number of lines to return from the top."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "content": "package main\n\nimport (\n...",
  "lines_returned": 10,
  "eof_reached": false
}
```

---

## 4. Tool: `tail` (Last N Lines)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the file."
    },
    "lines": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 10000,
      "description": "Number of lines to return from the end."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "content": "...last 10 lines...",
  "lines_returned": 10
}
```

### Implementation Note
Use a circular buffer (ring buffer) of size `args.Lines` when scanning line-by-line. This ensures memory usage is O(N lines) regardless of file size — critical for large log files.

---

## 5. Tool: `du` (Disk Usage)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the directory or file."
    },
    "max_depth": {
      "type": "integer",
      "default": 1,
      "minimum": 1,
      "maximum": 5,
      "description": "How deep to break down usage by subdirectory."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "total_bytes": 524288,
  "total_human": "512 KB",
  "breakdown": [
    { "path": "internal/", "bytes": 204800, "human": "200 KB" },
    { "path": "docs/",     "bytes": 319488, "human": "312 KB" }
  ]
}
```

### Human Readable Format
Format bytes to the nearest unit: `B`, `KB`, `MB`, `GB`. Use 1024 divisors (binary prefix).

---

## 6. Policy Rules

1. **Pure Go:** All four tools use stdlib only (`os`, `io`, `bufio`, `path/filepath`). No `os/exec`.
2. **Policy Evaluation:** Call `PolicyEngine.Evaluate(toolName, []string{args.Path}, false)` before any filesystem access.
3. **Symlinks:** Resolve with `filepath.EvalSymlinks` before policy check. Do not follow symlinks during walks.
4. **Binary file detection (`head`/`tail`):** Detect binary content (same check as `cat`) and return `EXEC_FAILED`.

---

## 7. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| `path` outside `allowed_root` | `ok: false`, `code: POLICY_DENIED` |
| `path` does not exist | `ok: false`, `code: NOT_FOUND` |
| `head`/`tail` on a directory | `ok: false`, `code: INVALID_ARGS` |
| `tree` on a file | `ok: true`, shows single entry |
| `tree` entries exceed `max_matches` | `ok: true`, `meta.truncated: true` |
| `head`/`tail` on binary file | `ok: false`, `code: EXEC_FAILED` |
| `du` on a single file | `ok: true`, breakdown has 1 entry |
