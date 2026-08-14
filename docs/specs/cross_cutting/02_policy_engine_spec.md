# Spec: Policy Engine
# Cross-cutting — gates every tool call in osmcp

## Status: APPROVED
## Version: 1.0

---

## 1. Purpose

The Policy Engine is the central safety gate. It runs **before** any tool executes and is
the primary trust boundary between an AI agent and the OS. All enforcement decisions flow
through a single `Evaluate()` call — no tool may bypass it.

**Design principle: Fail closed.** If the Policy Engine cannot make a definitive ALLOW
decision, it MUST deny.

---

## 2. Policy Config Format (TOML)

The policy is loaded from a TOML file at server startup. The file path is resolved in
this order (first match wins):

1. `--policy <path>` CLI flag
2. `OSMCP_POLICY_PATH` environment variable
3. `.osmcp/policy.toml` in the working directory
4. `~/.config/osmcp/policy.toml`
5. Built-in safe defaults (read-only, no run_script, allowed_root = cwd)

### Reference config

```toml
[policy]
# Absolute path. All file arguments are validated against this prefix.
# Symlinks are resolved before comparison.
allowed_root = "/home/user/myproject"

# Explicit tool allowlist. Only tools in this list are returned in tools/list.
# Any tool NOT listed here is invisible to the agent.
allowed_tools = [
  "grep", "find", "ls", "cat", "head", "tail",
  "wc", "stat",
  "git_status", "git_diff", "git_log",
  "sed", "jq", "diff"
]

# Mutating filesystem tools (cp, mv, rm, mkdir). Off by default.
allow_mutation = false

# Git write ops (git_add, git_commit, git_branch). Off by default.
allow_git_write = false

# Tier 2 run_script. DEFERRED TO v2. Must be false in v1.
allow_run_script = false

[limits]
# Per-call wall-clock timeout in milliseconds.
timeout_ms = 10000

# Maximum combined stdout+stderr size. Excess is truncated; meta.truncated = true.
max_output_bytes = 524288   # 512 KB

# Maximum match results (grep, find). Excess is truncated.
max_matches = 1000

[audit]
# Destination: "stderr" | "file"
destination = "stderr"

# Only used when destination = "file"
# path = "/var/log/osmcp/audit.jsonl"
```

---

## 3. Evaluation Algorithm

```
Evaluate(toolName string, pathArgs []string, isMutating bool) → error

1. Is toolName in policy.AllowedTools?
      NO  → return POLICY_DENIED ("tool not in allowlist")

2. For each path in pathArgs:
      a. Resolve symlinks → absPath
      b. Does absPath have policy.AllowedRoot as a prefix?
            NO → return POLICY_DENIED ("path outside allowed root: <absPath>")

3. If isMutating AND NOT policy.AllowMutation:
      → return POLICY_DENIED ("mutation not permitted by policy")

4. If toolName requires git write AND NOT policy.AllowGitWrite:
      → return POLICY_DENIED ("git write not permitted by policy")

5. → return nil (ALLOW)
```

### Notes

- Step 1 (tool allowlist) is evaluated **first**. A denied tool never reaches path checks.
- All symlinks in path arguments are resolved before prefix comparison. This prevents
  symlink escape attacks (e.g. `../../../etc` → resolves to `/etc` → denied).
- Resource limits (`timeout_ms`, `max_output_bytes`) are NOT enforced by the Policy Engine.
  They are enforced by the Execution Engine after the policy ALLOW decision.

---

## 4. Tool Visibility (tools/list filtering)

When the MCP client calls `tools/list`, osmcp returns ONLY the tools present in
`policy.AllowedTools`. Tools not in the list are **not returned** — they are invisible
to the agent. This is stronger than runtime denial: the agent cannot reason about or
attempt to call a tool it cannot see.

---

## 5. Policy Scope & Override Rules (v1)

| Scope | Source | Can expand? | Can restrict? |
|-------|--------|-------------|---------------|
| User-level | `~/.config/osmcp/policy.toml` | — | ✅ baseline |
| Project-level | `.osmcp/policy.toml` | ❌ | ✅ only restrict |
| Session | `--policy` CLI flag | ✅ | ✅ full override |

In v1, the `--policy` flag provides the active policy with no merging. Multi-level
merging is a v2 feature.

---

## 6. Policy Validation

Running `osmcp --validate` must:
- Parse the TOML file.
- Check that `allowed_root` is an absolute path that exists.
- Check that all entries in `allowed_tools` are known tool names.
- Check that `timeout_ms` is between 100 and 60000.
- Check that `max_output_bytes` is between 1024 and 10485760 (10 MB).
- Print a structured validation report to stdout.
- Exit 0 on success, exit 1 on any error.

---

## 7. Audit Log Entry (per call)

Every call through the Policy Engine — ALLOW or DENY — produces one NDJSON audit line:

```json
{
  "ts": "2026-08-14T10:30:00.000Z",
  "call_id": "c_xyz789",
  "tool": "rm",
  "path_args": ["<sanitized>"],
  "policy_decision": "DENIED",
  "denial_code": "POLICY_DENIED",
  "duration_ms": 1,
  "ok": false
}
```

> `path_args` values are logged as-is (not redacted) since osmcp operates on local files
> the operator controls. If a future multi-tenant mode is added, path redaction must be
> revisited.

---

## 8. Non-Goals

- This spec does NOT cover run_script binary blocklists (v2).
- This spec does NOT cover per-agent named profiles (v2).
- This spec does NOT cover hot-reload on SIGHUP (v2).
