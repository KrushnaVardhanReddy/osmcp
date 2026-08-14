# Spec: Response Envelope & Error Codes
# Cross-cutting — applies to every tool in osmcp

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

Every tool call in osmcp — whether a read, transform, or (future) mutation — returns a
single, uniform JSON envelope. This contract must never change without a version bump.
Agents parse `ok` first, then `data`, never tool-specific shapes.

---

## 2. Envelope Shape

```json
{
  "version": "1",
  "ok": true,
  "tool": "<tool-name>",
  "data": { },
  "meta": {
    "execution_time_ms": 0,
    "truncated": false
  },
  "error": null
}
```

### Field rules

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `version` | string | ✅ | Always `"1"` in v1. Increment on breaking envelope changes. |
| `ok` | bool | ✅ | `true` on success, `false` on any failure (policy, exec, args). |
| `tool` | string | ✅ | Exact name of the tool that produced this response. |
| `data` | object \| null | ✅ | Tool-specific payload on success. `null` on failure. |
| `meta` | object | ✅ | Always present. Contains timing and truncation info. |
| `meta.execution_time_ms` | int | ✅ | Wall-clock time from call receipt to response, in milliseconds. |
| `meta.truncated` | bool | ✅ | `true` if output was capped by `max_output_bytes` or `max_matches`. |
| `error` | object \| null | ✅ | `null` on success. Structured error object on failure. |

---

## 3. Error Object Shape

```json
{
  "code": "POLICY_DENIED",
  "message": "Path '/etc/passwd' is outside the allowed root.",
  "retryable": false
}
```

| Field | Type | Notes |
|-------|------|-------|
| `code` | string (enum) | Stable machine-readable code. Never a free-text string. |
| `message` | string | Human-readable explanation. May change between versions. |
| `retryable` | bool | Hint to the agent: is retrying with modified args likely to succeed? |

---

## 4. Error Code Enum (Stable)

| Code | Meaning | Retryable |
|------|---------|-----------|
| `POLICY_DENIED` | Policy Engine rejected the call (path/tool/mutation not permitted) | No |
| `INVALID_ARGS` | Arguments failed JSON schema validation | No |
| `NOT_FOUND` | Target file/path/resource does not exist | No |
| `TIMEOUT` | Execution exceeded the configured time limit | Sometimes — retry with narrower scope |
| `OUTPUT_TOO_LARGE` | Result exceeded the output size cap | Sometimes — retry with tighter query |
| `EXEC_FAILED` | Underlying command returned non-zero exit or runtime error | Depends on cause |

> **Rule:** No new codes may be added without a minor version bump. Agents MAY treat unknown
> codes as non-retryable `EXEC_FAILED`.

---

## 5. Success Example (grep)

```json
{
  "version": "1",
  "ok": true,
  "tool": "grep",
  "data": {
    "matches": [
      { "file": "src/main.go", "line": 42, "text": "// TODO: fix this" }
    ],
    "count": 1
  },
  "meta": {
    "execution_time_ms": 12,
    "truncated": false
  },
  "error": null
}
```

---

## 6. Failure Example (policy denied)

```json
{
  "version": "1",
  "ok": false,
  "tool": "rm",
  "data": null,
  "meta": {
    "execution_time_ms": 2,
    "truncated": false
  },
  "error": {
    "code": "POLICY_DENIED",
    "message": "Path '/etc/passwd' is outside the allowed root '/home/user/project'.",
    "retryable": false
  }
}
```

---

## 7. Implementation Requirements

- The Go package `internal/response` MUST be the only place envelopes are constructed.
- No tool may construct its own JSON response directly — all must use `response.Success()` and `response.Failure()`.
- `meta.execution_time_ms` must be measured from **before** policy evaluation to **after** result serialization.
- `meta.truncated` must be set to `true` whenever any output is cut short, regardless of the reason.

---

## 8. Non-Goals

- This spec does NOT define tool-specific `data` shapes. Each tool has its own spec.
- This spec does NOT cover streaming responses (v2).
