# Spec: File Inspection Tools (ls, cat, stat, wc)
# Phase 1 — Read-only filesystem exploration

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Provide a safe, capability-bound way for agents to explore the local filesystem and read file contents. These tools must be implemented in **Pure Go** using the standard library (`os`, `io`, `path/filepath`) to ensure zero shell-injection risk and perfect cross-platform compatibility.

---

## 2. Tool: `ls` (List Directory)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the directory to list."
    },
    "recursive": {
      "type": "boolean",
      "default": false,
      "description": "If true, walks subdirectories."
    },
    "max_depth": {
      "type": "integer",
      "default": 1,
      "minimum": 1,
      "maximum": 10,
      "description": "Maximum depth for recursive listing."
    },
    "show_hidden": {
      "type": "boolean",
      "default": false,
      "description": "If true, includes files and directories starting with a dot (.)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "entries": [
    {
      "name": "main.go",
      "path": "/src/main.go",
      "is_dir": false,
      "size": 1024,
      "mod_time": "2026-08-14T10:00:00Z",
      "mode": "-rw-r--r--"
    }
  ],
  "count": 1
}
```

### Truncation
If the number of entries reaches the policy `max_matches`, stop walking, set `meta.truncated = true`, and return the collected entries.

---

## 3. Tool: `cat` (Read File)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the file to read."
    },
    "start_line": {
      "type": "integer",
      "default": 1,
      "minimum": 1,
      "description": "Line number to start reading from (1-indexed)."
    },
    "end_line": {
      "type": "integer",
      "description": "Line number to stop reading at (inclusive). If omitted, reads to EOF."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "content": "package main\n\nfunc main() {}",
  "lines_returned": 3,
  "eof_reached": true
}
```

### Truncation
Stop reading and set `meta.truncated = true` if the output exceeds the policy `max_output_bytes`.

---

## 4. Tool: `stat` (File Metadata)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the file or directory."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "name": "src",
  "path": "/project/src",
  "is_dir": true,
  "size": 4096,
  "mod_time": "2026-08-14T10:00:00Z",
  "mode": "drwxr-xr-x"
}
```

---

## 5. Tool: `wc` (Word/Line Count)

### Input Schema
```json
{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute path to the file."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "lines": 142,
  "words": 890,
  "bytes": 5600
}
```

---

## 6. Execution & Policy Rules

1. **Pure Go:** All four tools must use `os`, `io`, `bufio`, and `filepath` packages exclusively. `os/exec` is strictly forbidden.
2. **Policy Evaluation:** 
   - `PolicyEngine.Evaluate(toolName, []string{args.Path}, false)` must be called before opening any file or directory.
3. **Symlinks:** Symlinks must be resolved (`filepath.EvalSymlinks`) before policy evaluation, as defined in the Policy Engine spec.
4. **Binary Files (`cat`):** `cat` must detect binary files (e.g., by checking if the first 512 bytes contain null bytes or using `http.DetectContentType`) and return an `EXEC_FAILED` error indicating "Cannot read binary file as text" to prevent terminal garbage.

---

## 7. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| `path` outside `allowed_root` | `ok: false`, `code: POLICY_DENIED` |
| `path` does not exist | `ok: false`, `code: NOT_FOUND` |
| `cat` on a directory | `ok: false`, `code: INVALID_ARGS` |
| `ls` on a file | `ok: true`, returns 1 entry for the file itself |
| `cat` hits `max_output_bytes` | `ok: true`, `meta.truncated: true`, `eof_reached: false` |
| `ls` hits `max_matches` | `ok: true`, `meta.truncated: true` |
| `cat` on binary file | `ok: false`, `code: EXEC_FAILED` ("Cannot read binary file") |
