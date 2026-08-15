# Spec: Data Transformation Tools (jq, sed, diff)
# Phase 1 — Read-only data transformation

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Provide safe, in-process data transformation tools for AI agents. These tools transform data passed as input strings — they do not write to any file. All are implemented in **Pure Go** using curated third-party libraries. `os/exec` is strictly forbidden.

| Tool | Library | Description |
|------|---------|-------------|
| `jq` | `github.com/itchyny/gojq` | Query and filter JSON |
| `sed` | Go `regexp` stdlib | Stream editing and regex replace |
| `diff` | `github.com/sergi/go-diff` | Unified diff between two strings |

---

## 2. Tool: `jq` (JSON Query)

### Input Schema
```json
{
  "type": "object",
  "required": ["input", "filter"],
  "properties": {
    "input": {
      "type": "string",
      "description": "JSON string to query."
    },
    "filter": {
      "type": "string",
      "description": "A jq filter expression (e.g. '.users[] | .name')."
    },
    "compact": {
      "type": "boolean",
      "default": false,
      "description": "If true, output is compact (no pretty-print)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "result": "[\"Alice\", \"Bob\"]",
  "output_type": "array"
}
```

### Error Handling
- Invalid JSON input → `ok: false`, `code: INVALID_ARGS` ("Invalid JSON input")
- Invalid jq filter syntax → `ok: false`, `code: INVALID_ARGS` ("Invalid jq filter: <err>")

---

## 3. Tool: `sed` (Stream Edit)

### Input Schema
```json
{
  "type": "object",
  "required": ["input", "expression"],
  "properties": {
    "input": {
      "type": "string",
      "description": "The text content to transform."
    },
    "expression": {
      "type": "string",
      "description": "A sed-style substitute expression: s/pattern/replacement/flags. Supported flags: g (global), i (case-insensitive)."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "result": "hello world",
  "replacements_made": 3
}
```

### Expression Parsing
The expression MUST match the form `s/pattern/replacement/flags`.
- The delimiter is always `/`.
- Pattern is compiled using Go's `regexp.Compile` (RE2 syntax).
- Invalid regex → `ok: false`, `code: INVALID_ARGS`.
- Invalid expression format → `ok: false`, `code: INVALID_ARGS`.

### Truncation
If output exceeds `policy.Limits().MaxOutputBytes`, truncate and set `meta.truncated = true`.

---

## 4. Tool: `diff` (Text Comparison)

### Input Schema
```json
{
  "type": "object",
  "required": ["a", "b"],
  "properties": {
    "a": {
      "type": "string",
      "description": "The original text (left side)."
    },
    "b": {
      "type": "string",
      "description": "The modified text (right side)."
    },
    "context_lines": {
      "type": "integer",
      "default": 3,
      "minimum": 0,
      "maximum": 10,
      "description": "Number of context lines around each change."
    }
  }
}
```

### Output Shape (`data` field)
```json
{
  "patch": "@@ -1,4 +1,4 @@\n hello\n-world\n+Go\n end",
  "additions": 1,
  "deletions": 1,
  "identical": false
}
```

### Truncation
If patch output exceeds `policy.Limits().MaxOutputBytes`, truncate and set `meta.truncated = true`.

---

## 5. Policy Rules

1. **No file I/O:** All three tools operate purely on strings passed as arguments. They are NOT subject to `allowed_root` path checks because they do not read from disk.
2. **Policy Evaluation:** Call `PolicyEngine.Evaluate(toolName, []string{}, false)` to check tool visibility only.
3. **Output Limits:** Respect `policy.Limits().MaxOutputBytes` for `sed` and `diff`.
4. **Input Limits:** If `input` or `a`/`b` strings exceed `policy.Limits().MaxOutputBytes`, return `INVALID_ARGS` before processing.

---

## 6. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| `jq` filter is invalid syntax | `ok: false`, `code: INVALID_ARGS` |
| `jq` input is not valid JSON | `ok: false`, `code: INVALID_ARGS` |
| `sed` expression has wrong format | `ok: false`, `code: INVALID_ARGS` |
| `sed` pattern is invalid regex | `ok: false`, `code: INVALID_ARGS` |
| `diff` on identical strings | `ok: true`, `identical: true`, `patch: ""` |
| Output exceeds `MaxOutputBytes` | `ok: true`, `meta.truncated: true` |
