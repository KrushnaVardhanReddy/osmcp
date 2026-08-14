# Spec: grep Tool
# Phase 1 — First typed tool; validates the full MCP → Policy → Executor → Envelope stack

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

`grep` is the first tool implemented in osmcp. Its primary role is to validate the full
vertical stack: MCP wire format → Policy Engine → Execution Engine → Response Envelope.
It must be implemented and passing tests before any other tool is started.

---

## 2. MCP Tool Name

```
grep
```

---

## 3. Input Schema

```json
{
  "type": "object",
  "required": ["pattern", "path"],
  "properties": {
    "pattern": {
      "type": "string",
      "description": "Search pattern. Treated as a regular expression unless literal=true."
    },
    "path": {
      "type": "string",
      "description": "Absolute path to a file or directory to search."
    },
    "recursive": {
      "type": "boolean",
      "default": true,
      "description": "Search subdirectories recursively. Ignored if path is a file."
    },
    "case_sensitive": {
      "type": "boolean",
      "default": true,
      "description": "If false, performs a case-insensitive search (-i flag)."
    },
    "literal": {
      "type": "boolean",
      "default": false,
      "description": "If true, treats pattern as a fixed string, not a regex (-F flag)."
    },
    "context_lines": {
      "type": "integer",
      "default": 0,
      "minimum": 0,
      "maximum": 10,
      "description": "Number of lines of context to include before and after each match (-C flag)."
    },
    "include": {
      "type": "string",
      "description": "Glob pattern to restrict searched files. Example: '*.go'. Applied via --include flag."
    },
    "exclude": {
      "type": "string",
      "description": "Glob pattern to exclude files from search. Example: '*_test.go'."
    },
    "max_matches": {
      "type": "integer",
      "description": "Override the policy max_matches limit for this call. Cannot exceed policy limit."
    }
  }
}
```

---

## 4. Execution

The tool executes `grep` directly via `os/exec` with `shell: false`. Arguments are built
from the schema fields — no shell string interpolation at any point.

### Argument construction

```
grep
  [-r]                         if recursive=true and path is a directory
  [-i]                         if case_sensitive=false
  [-F]                         if literal=true
  [-C <context_lines>]         if context_lines > 0
  [--include=<include>]        if include is set
  [--exclude=<exclude>]        if exclude is set
  [-m <max_matches>]           always set to min(args.max_matches, policy.max_matches)
  --line-number
  --with-filename
  --null                       (to safely handle filenames with special chars)
  -- <pattern> <path>          pattern and path always come last, separated by --
```

### PATH resolution

The `grep` binary is resolved from the OS `PATH` at server startup. If `grep` is not
found, osmcp exits with a clear error message. The resolved path is stored and used for
all subsequent calls.

---

## 5. Output Shape (`data` field)

```json
{
  "matches": [
    {
      "file": "src/main.go",
      "line": 42,
      "text": "// TODO: fix this",
      "context_before": [],
      "context_after": []
    }
  ],
  "count": 1
}
```

| Field | Type | Notes |
|-------|------|-------|
| `matches` | array | Ordered list of match objects. Empty array if no matches (not null). |
| `matches[].file` | string | Absolute path to the file containing the match. |
| `matches[].line` | int | 1-indexed line number of the match. |
| `matches[].text` | string | The matched line, trimmed of trailing newline. |
| `matches[].context_before` | string[] | Lines before the match (empty if context_lines=0). |
| `matches[].context_after` | string[] | Lines after the match (empty if context_lines=0). |
| `count` | int | Total number of matches. Equals `len(matches)` unless truncated. |

### Truncation

If the number of matches reaches `max_matches`, the tool stops collecting further matches,
sets `meta.truncated = true`, and returns what was collected. `count` reflects the
truncated count, not the total possible matches.

---

## 6. Policy Checks

Before execution, the Policy Engine evaluates:

1. `"grep"` is in `policy.AllowedTools` → else `POLICY_DENIED`
2. `path` (after symlink resolution) is within `policy.AllowedRoot` → else `POLICY_DENIED`
3. `grep` is read-only → no mutation check required

---

## 7. Behaviour Matrix

| Scenario | Expected response |
|----------|-----------------|
| Pattern found, single file | `ok: true`, matches array populated |
| Pattern not found | `ok: true`, `matches: []`, `count: 0` |
| `path` is a file (not dir) | `ok: true`, searches only that file |
| `path` does not exist | `ok: false`, `code: NOT_FOUND` |
| `path` outside `allowed_root` | `ok: false`, `code: POLICY_DENIED` |
| `grep` not in `allowed_tools` | `ok: false`, `code: POLICY_DENIED` |
| Execution exceeds `timeout_ms` | `ok: false`, `code: TIMEOUT` |
| Matches exceed `max_matches` | `ok: true`, `meta.truncated: true`, partial results |
| Invalid regex pattern | `ok: false`, `code: EXEC_FAILED`, grep stderr in message |
| Empty `pattern` | `ok: false`, `code: INVALID_ARGS` |
| Empty `path` | `ok: false`, `code: INVALID_ARGS` |

---

## 8. Unit Test Requirements

All tests live in `internal/tools/grep_test.go`.

- ✅ Match found in single file
- ✅ Match found recursively across multiple files
- ✅ No matches → empty array (not error)
- ✅ case_sensitive=false finds match regardless of case
- ✅ literal=true does not treat `.` as regex wildcard
- ✅ context_lines=2 returns 2 lines before and after
- ✅ include="*.go" filters to only Go files
- ✅ max_matches=1 truncates at 1 result, meta.truncated=true
- ✅ Path outside allowed_root → POLICY_DENIED
- ✅ grep not in allowed_tools → POLICY_DENIED
- ✅ Non-existent path → NOT_FOUND
- ✅ Timeout simulation → TIMEOUT

---

## 9. E2E Test Requirements

Tests in `e2e/grep_e2e_test.go` spin up a real osmcp process over stdio and send
actual MCP JSON-RPC calls. Fixtures in `testdata/fixtures/`.

- ✅ Full round-trip: call grep via MCP stdio → assert uniform envelope response
- ✅ Policy denial round-trip: path outside root → assert POLICY_DENIED envelope
- ✅ tools/list with read-only policy → grep is visible
- ✅ tools/list with grep excluded from policy → grep is NOT in the list

---

## 10. Non-Goals

- This spec does NOT cover `git grep`. Use the `grep` tool against a git worktree.
- This spec does NOT cover binary file searching (grep's default skip applies).
- This spec does NOT cover streaming output (v2).
