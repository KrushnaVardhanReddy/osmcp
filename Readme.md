# osmcp

A typed, policy-controlled OS capability layer for AI agents.

## Why

AI agents need to interact with real developer environments: read files, search code, modify files, inspect Git, run tests, execute build commands, transform data, inspect project state. Giving them unrestricted shell access is powerful but risky.

`osmcp` provides a controlled middle layer.

Instead of:

```
Agent → raw shell → OS
```

`osmcp` provides:

```
Agent
  ↓
MCP
  ↓
Typed Tools
  ↓
Policy Engine
  ↓
Execution Engine
  ↓
OS
```

Common operations are exposed as structured MCP tools. Anything else can use a restricted script execution capability. Security is based primarily on **capabilities and execution boundaries**, rather than command blocklists.

The long-term goal is not to wrap Unix commands. The goal is to provide a reliable local OS capability runtime for AI agents.

## Design

### Architecture layers

- **Typed Tools** — each common operation (grep, sed, git status, etc.) is exposed as its own MCP tool with a proper JSON schema. **All Tier 1 tools are implemented in Pure Go** (either via stdlib or mature third-party Go libraries like `gojq` or `go-git`). This guarantees true cross-platform portability without depending on host OS binaries (GNU vs BSD differences).
- **Policy Engine** — the central gate. Decides which capabilities an agent has access to (which tools are even visible), and enforces boundaries per call: allowed paths, resource limits, mutation permissions.
- **Execution Engine** — reserved strictly for the v2 `run_script` tool. Runs arbitrary shell scripts via `exec()` with `shell:false` and deep sandboxing.

### Two tiers of tools

**Tier 1 — Typed Tools**
Structured, schema-defined tools covering the common 90% of file/text/git operations. Safe by construction — implemented natively in Go, typed args, structured results. No subprocess execution vulnerabilities.

**Tier 2 — `run_script` (restricted execution capability)**
Accepts a full shell script body (bash/sh) for pipelines and anything not covered by Tier 1 (e.g. `grep -l TODO **/*.py | xargs wc -l | sort -n`). Still gated by the Policy Engine and sandboxed via:
- Path jail (restricted to a configured root directory)
- Execution timeout
- Output size cap
- Runs as an unprivileged user (no elevated privileges)

Blocklisting dangerous patterns is treated as a weak secondary defense — the real safety boundary is *scope and capability grants* (filesystem jail, no root, resource limits, policy-gated visibility), not string matching.

### Why Go

- Official MCP SDKs exist for Go with solid stdio/JSON-RPC transport support.
- Compiles to a single static binary per platform — trivial Homebrew formula, zero runtime dependencies. Because tools are implemented natively in Go, the binary doesn't even depend on the host having `grep`, `git`, or `jq` installed.
- Goroutines make per-call timeouts and cancellation (needed for `run_script`) straightforward.

Rust remains a solid alternative if deeper OS-level sandboxing (seccomp, Linux namespaces, landlock) becomes a priority later — but that would be a rewrite of the execution layer only, not the whole tool surface.

## Planned Tool Set (Typed Tools layer)

### File search & inspection
- `grep` — pattern search (regex, recursive, case-insensitive, context lines, file filtering)
- `find` — locate files by name/type/size/modified-time
- `ls` — list directory contents (detail/hidden-file flags)
- `cat` / `head` / `tail` — read file contents (full, first N, last N lines)
- `wc` — count lines/words/bytes
- `file` — detect file type

### Text transformation
- `sed` — find/replace, line-based editing
- `awk` — field extraction, simple transforms
- `sort` — sort lines (numeric, reverse, by field)
- `uniq` — dedupe (with count option)
- `diff` — compare two files/directories
- `jq` — JSON querying/transformation

### Filesystem operations
- `cp` / `mv` / `rm` — copy, move, delete (path-jail enforced; `rm` should require explicit confirmation semantics)
- `mkdir` — create directories
- `touch` — create/update file timestamps
- `chmod` / `chown` — **disabled by default**, opt-in flag required (privilege-escalation risk)

### Git operations
- `git_status`, `git_diff`, `git_log` — read-only, safe by default
- `git_add`, `git_commit`, `git_branch` — mutating, local-only
- `git_push` / `git_pull` / `git_reset --hard` — **excluded from default set**; touches remote state or destroys history. Available only via `run_script` if explicitly needed.

### Metadata / system
- `pwd` — current directory context
- `du` / `df` — disk usage, free space
- `stat` — file metadata (size, timestamps, permissions)

### Tier 2
- `run_script` — sandboxed shell script execution (bash/sh) for pipelines and anything not covered above

## Response Format

Every tool — Tier 1 typed tools and the Tier 2 `run_script` — returns the same uniform JSON envelope, regardless of which tool was called. This is deliberate: agents should never need a tool-specific parser to know whether a call succeeded, what went wrong, or where the actual payload lives.

### Shape

```json
{
  "ok": true,
  "tool": "grep",
  "data": { },
  "meta": {
    "execution_time_ms": 0,
    "truncated": false
  },
  "error": null
}
```

- `ok` — top-level boolean. Lets the agent branch immediately without parsing exit codes or scanning stderr text.
- `tool` — name of the tool that produced this response.
- `data` — tool-specific payload (see examples below). Always lives under this same key so downstream logic can generically inspect `response.data` without knowing which tool produced it. `null` on failure.
- `meta` — cross-cutting metadata common to every tool: timing, truncation flags, etc. Kept separate from `data` so the payload itself stays clean.
- `error` — `null` on success. On failure, an object with a **stable error code** (not a raw message the model has to pattern-match), a human-readable message, and whether the operation is safely retryable.

### Success example

```json
{
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

### Failure example

```json
{
  "ok": false,
  "tool": "rm",
  "data": null,
  "meta": {
    "execution_time_ms": 2
  },
  "error": {
    "code": "POLICY_DENIED",
    "message": "Path '/etc/passwd' is outside the allowed root.",
    "retryable": false
  }
}
```

### `run_script` (Tier 2) payload

Since script output is inherently unstructured, its `data` object carries the raw execution result — but still wrapped in the same envelope, so agents don't need a special-case parser for Tier 2 vs Tier 1:

```json
{
  "stdout": "...",
  "stderr": "...",
  "exit_code": 0
}
```

### Error code enum (initial set)

| Code | Meaning | Retryable |
|---|---|---|
| `POLICY_DENIED` | Policy Engine rejected the call (path/command/mutation not permitted) | No |
| `INVALID_ARGS` | Arguments failed schema validation | No |
| `NOT_FOUND` | Target file/path/resource doesn't exist | No |
| `TIMEOUT` | Execution exceeded the configured time limit | Sometimes (narrower scope) |
| `OUTPUT_TOO_LARGE` | Result exceeded the output size cap (see `meta.truncated`) | Sometimes (narrower query) |
| `EXEC_FAILED` | Underlying command/process failed (non-zero exit, runtime error) | Depends on cause |

Keeping error codes as a stable enum (rather than free-text messages) lets an agent react programmatically — e.g. retry with a narrower path on `POLICY_DENIED`, or retry with tighter scope on `TIMEOUT` — instead of having to interpret prose.

## Build Priority (v1)

1. **Core read tools** — `grep`, `find`, `ls`, `cat`
2. **Read-only git** — `git_status`, `git_diff`, `git_log`
3. **Core transform tools** — `sed`, `jq`, `diff`
4. **Fallback** — `run_script`
5. **Mutating operations** — `cp`, `mv`, `rm`, `mkdir`, `chmod`, git write ops (require extra safety design: dry-run mode, confirmation flags, strict path-jail enforcement)

## Open Questions

- **Policy Engine shape**: declarative config file (allowed paths/commands/limits) vs. a programmable rules engine or plugin system.
- **Policy scope**: gate execution only (all tools visible, calls rejected at runtime if disallowed) vs. capability-based tool visibility (agent only ever sees tools the policy grants).
- Sandboxing depth for `run_script`: host-level path restriction only, vs. per-call container/sandbox isolation.
- Whether `chmod`/`chown` ship at all in v1, or are deferred entirely.
- Confirmation/dry-run UX for destructive operations (`rm`, `git reset --hard` via script).

## Distribution

- Compiled static binary per platform (macOS/Linux, arm64/amd64).
- Installed via `brew install osmcp` (Homebrew formula pulling from GitHub releases, or a `brew tap`).
- Runs as an MCP server over stdio, addable to any MCP-compatible client config.

---

## Threat Model

osmcp is a local-process MCP server. Its safety guarantees are scoped accordingly.

**What osmcp protects against:**

- **Agent going off-rails** — an AI agent autonomously performing destructive or out-of-scope operations (deleting files, resetting git history, exfiltrating data via shell).
- **Prompt injection** — attacker-controlled content in a file or command output tricking the agent into issuing a dangerous tool call.
- **Scope creep** — an agent accessing paths, repos, or system resources outside what the human operator intended.
- **Accidental mutation** — an agent that should only be reading code accidentally writing or deleting files.

**What osmcp does NOT protect against:**

- A compromised or malicious osmcp binary itself.
- A human operator who deliberately configures a permissive policy.
- Kernel-level exploits from within a `run_script` sandbox (host-level isolation only in v1).
- Network exfiltration if network access is not explicitly blocked at the OS level or policy level.

**Trust boundary:** The human operator (the person who installs and configures osmcp) is trusted. The AI agent is untrusted by default — it receives only the capabilities the operator explicitly grants.

---

## Policy Engine

The Policy Engine is the central gate for every tool call. It runs before any tool executes and is the primary safety boundary.

### Design decision: capability-based tool visibility

Agents only see tools their policy grants. A tool not in the granted set is invisible — it does not appear in the MCP tool list, so the agent cannot reason about or attempt to call it. Runtime rejection (call arrives, then gets denied) is a secondary fallback, not the primary model.

This is stronger than a blocklist: you cannot be prompt-injected into calling a tool you cannot see.

### Policy config format

Policy lives in a TOML file. The server loads it at startup. Hot-reload on `SIGHUP` is planned for v2.

```toml
# ~/.config/osmcp/policy.toml  (default location)

[policy]
# Filesystem root the agent is jailed to. All path arguments are validated
# against this prefix before execution. Symlinks are resolved first.
allowed_root = "/home/user/myproject"

# Explicit tool allowlist. Only these tools appear in the MCP tool listing.
# Omit a tool to hide it entirely from the agent.
allowed_tools = [
  "grep", "find", "ls", "cat", "head", "tail",
  "git_status", "git_diff", "git_log",
  "sed", "jq", "diff"
]

# Allow mutating filesystem tools (cp, mv, rm, mkdir).
# false by default — must be explicitly opted in.
allow_mutation = false

# Allow git write operations (git_add, git_commit, git_branch).
allow_git_write = false

# Allow Tier 2 run_script. Disabled by default.
allow_run_script = false

[limits]
# Per-call execution timeout.
timeout_ms = 10000

# Maximum stdout+stderr bytes returned per call.
# Excess is truncated and flagged via meta.truncated = true.
max_output_bytes = 524288   # 512 KB

# Maximum number of grep/find matches returned before truncation.
max_matches = 1000

[run_script]
# Only meaningful if allow_run_script = true.
# Hard-block these binaries inside any run_script call, regardless of policy.
# Network tools are blocked by default to prevent exfiltration.
blocked_binaries = ["curl", "wget", "nc", "ssh", "scp", "rsync", "python3", "node"]

# Whether to allow outbound network from run_script.
# Requires OS-level enforcement (seccomp/landlock) — planned for v2.
allow_network = false
```

### Policy evaluation order

```
Incoming tool call
  │
  ├─ Is this tool in allowed_tools? ────── No  ──► POLICY_DENIED (invisible tool)
  │
  ├─ Is the path within allowed_root? ──── No  ──► POLICY_DENIED
  │
  ├─ Is this a mutating call?
  │    └─ Is allow_mutation = true? ─────── No  ──► POLICY_DENIED
  │
  ├─ Resource limits check (timeout, output cap)
  │
  └─ Pass to Execution Engine
```

### Policy scopes (planned)

| Scope | File location | Purpose |
|-------|--------------|---------|
| Global default | `~/.config/osmcp/policy.toml` | User-level baseline |
| Project override | `.osmcp/policy.toml` (repo root) | Per-repo tightening |
| Session grant | CLI flag `--policy ./custom.toml` | One-off overrides |

Project-level policy can only *restrict*, never *expand* beyond the global default. This prevents a malicious repo from widening its own permissions.

---

## Agent Identity & Session Model

**v1 — Stateless, single-agent, stdio only**

Each osmcp process handles one MCP client over stdio. The agent identity is implicit — it's whoever launched the process. No auth token required because the trust boundary is the OS process model (same user, local machine).

**v2 — Multi-agent, HTTP/SSE transport**

When an HTTP transport mode is added, osmcp will need:

- A named **profile** per agent (maps to a policy config section).
- A **session token** (Bearer token or mTLS client cert) to identify which profile applies.
- Concurrent session handling — each session gets its own isolated policy evaluation context.

```toml
# Multi-agent example (v2 shape, not yet implemented)
[agents.readonly-bot]
allowed_tools = ["grep", "find", "ls", "cat"]
allowed_root = "/home/user/project"

[agents.refactor-bot]
allowed_tools = ["grep", "sed", "git_status", "git_diff", "git_add", "git_commit"]
allowed_root = "/home/user/project"
allow_git_write = true
```

---

## `run_script` Hard Limits

Even with `allow_run_script = true`, the following limits are unconditional and cannot be overridden by policy config:

| Limit | Value | Rationale |
|-------|-------|-----------|
| Execution timeout | Configurable, max 60s hard cap | Prevent runaway processes |
| Output size | Configurable, max 10 MB hard cap | Prevent memory exhaustion |
| Working directory | Locked to `allowed_root` | Cannot `cd` outside jail |
| Privileged syscalls | Blocked (no `sudo`, `su`, `setuid`) | No privilege escalation |
| Network (v1) | Policy-configurable; blocked by default | Data exfiltration prevention |
| Network (v2) | Enforced via seccomp/landlock | Kernel-level guarantee |

**Default blocked binaries** (always blocked unless explicitly removed from `blocked_binaries`):

`curl`, `wget`, `nc` (netcat), `ssh`, `scp`, `rsync`, `nmap`, `socat`, `telnet`

The agent receives a `POLICY_DENIED` error with `code: "SCRIPT_BLOCKED_BINARY"` if any blocked binary is invoked.

---

## Audit Log

Every tool call — whether it succeeds, fails, or is denied by policy — produces a structured audit log entry. This is non-optional and always active.

### Format

One JSON object per line (`application/x-ndjson`), written to the configured audit log file (default: `stderr`).

```json
{
  "ts": "2026-08-14T10:30:00.000Z",
  "session_id": "s_abc123",
  "call_id": "c_xyz789",
  "tool": "rm",
  "args": { "path": "/etc/passwd" },
  "policy_decision": "DENIED",
  "denial_reason": "POLICY_DENIED",
  "duration_ms": 1,
  "ok": false
}
```

### Configuration

```toml
[audit]
# "stderr" (default), "file", or "syslog"
destination = "file"
path = "/var/log/osmcp/audit.jsonl"

# Rotate when file exceeds this size.
rotate_max_bytes = 10485760   # 10 MB
rotate_keep = 5
```

### What is logged

| Event | Logged |
|-------|--------|
| Tool call received | ✅ Tool name + sanitized args |
| Policy decision | ✅ ALLOW or DENY + reason code |
| Execution result | ✅ ok/fail + duration |
| Policy config load | ✅ File path + hash |
| Server start/stop | ✅ With version + config path |
| Raw stdout/stderr of `run_script` | ❌ (too large; only exit code logged) |

---

## Observability

### Metrics (v2)

osmcp will optionally expose a Prometheus-compatible metrics endpoint when running in HTTP transport mode.

| Metric | Type | Description |
|--------|------|-------------|
| `osmcp_calls_total` | Counter | Total tool calls, labelled by `tool` and `status` |
| `osmcp_call_duration_ms` | Histogram | Per-tool latency |
| `osmcp_policy_denials_total` | Counter | Policy denials, labelled by `denial_reason` |
| `osmcp_script_duration_ms` | Histogram | `run_script` execution times |

### Health endpoint (v2)

`GET /healthz` — returns `200 OK` with `{"status":"ok","version":"..."}` when the server is running and policy config is valid.

---

## Tool Discovery

Agents discover available tools via the standard MCP `tools/list` call. osmcp filters the response to only include tools the active policy grants — tools outside the allowed set are not returned at all.

Each tool descriptor includes a full JSON schema for its arguments, allowing agents to introspect parameter names, types, and constraints without documentation.

### Example `tools/list` response (read-only policy)

```json
{
  "tools": [
    {
      "name": "grep",
      "description": "Search for a pattern in files.",
      "inputSchema": {
        "type": "object",
        "required": ["pattern", "path"],
        "properties": {
          "pattern":        { "type": "string" },
          "path":           { "type": "string" },
          "recursive":      { "type": "boolean", "default": true },
          "case_sensitive": { "type": "boolean", "default": true },
          "context_lines":  { "type": "integer", "default": 0 },
          "include":        { "type": "string", "description": "Glob filter, e.g. '*.go'" }
        }
      }
    }
  ]
}
```

---

## Dry-Run & Confirmation Mode

Mutating tools (`rm`, `cp`, `mv`, `git_commit`, etc.) support a `dry_run` flag. When set, the tool simulates the operation and returns what *would* have happened, without making any changes.

```json
// Request
{ "tool": "rm", "args": { "path": "src/old_module/", "recursive": true, "dry_run": true } }

// Response
{
  "ok": true,
  "tool": "rm",
  "data": {
    "dry_run": true,
    "would_delete": [
      "src/old_module/handler.go",
      "src/old_module/handler_test.go",
      "src/old_module/types.go"
    ],
    "count": 3
  },
  "meta": { "execution_time_ms": 3, "truncated": false },
  "error": null
}
```

**Human-in-the-loop confirmation (v2):** An optional policy flag `require_human_confirm = true` for destructive operations will cause osmcp to pause and prompt the terminal user before executing. The agent receives a pending state response and must poll or await a webhook callback.

---

## Streaming / Long-Running Commands

`run_script` commands can run for seconds or minutes (e.g. `cargo build`, `npm install`, test suites). osmcp supports incremental output streaming via MCP's streaming response protocol.

**v1 — Buffered output**
Full stdout/stderr are buffered in memory (up to `max_output_bytes`) and returned as a single response. Simple, but not suitable for long-running commands.

**v2 — Streamed output**
osmcp will emit incremental MCP progress notifications as the script produces output. The final response is still the same uniform JSON envelope. Streaming is opt-in per call:

```json
{ "tool": "run_script", "args": { "script": "cargo build", "stream": true } }
```

If the MCP client does not support streaming, osmcp falls back to buffered mode automatically.

**Cancellation:** The agent (or human) can send an MCP `tools/cancel` call to terminate a running script. osmcp sends `SIGTERM` to the process group, waits 2s, then `SIGKILL`.

---

## Configuration

### Config file search order

osmcp looks for its policy config in this order (first match wins):

1. `--policy <path>` CLI flag
2. `OSMCP_POLICY_PATH` environment variable
3. `.osmcp/policy.toml` in the current working directory (project-level)
4. `~/.config/osmcp/policy.toml` (user-level, XDG-compliant)
5. Built-in defaults (read-only, no `run_script`, `allowed_root = cwd`)

### CLI flags

```
osmcp [flags]

Flags:
  --policy <path>       Path to policy TOML config file
  --audit-log <path>    Path to audit log file (default: stderr)
  --transport <mode>    Transport mode: stdio (default) | http
  --port <port>         HTTP port (only with --transport http, default: 8080)
  --version             Print version and exit
  --validate            Validate policy config and exit (no server started)
```

---

## Client Config Examples

### Claude Desktop

```json
{
  "mcpServers": {
    "osmcp": {
      "command": "osmcp",
      "args": ["--policy", "~/.config/osmcp/policy.toml"]
    }
  }
}
```

### Cursor

```json
{
  "mcpServers": {
    "osmcp": {
      "command": "osmcp",
      "args": ["--policy", ".osmcp/policy.toml"]
    }
  }
}
```

### VS Code (Copilot MCP extension)

```json
{
  "mcp.servers": {
    "osmcp": {
      "command": "osmcp",
      "args": ["--policy", "${workspaceFolder}/.osmcp/policy.toml"]
    }
  }
}
```

### Project-level (`.osmcp/policy.toml` committed to repo)

```toml
# .osmcp/policy.toml
# This file defines what any AI agent gets to do in this repo.
# Commit it — treat it like .editorconfig or .eslintrc.

[policy]
allowed_root = "."
allowed_tools = ["grep", "find", "ls", "cat", "head", "tail", "git_status", "git_diff", "git_log", "sed", "jq"]
allow_mutation = false
allow_run_script = false

[limits]
timeout_ms = 10000
max_output_bytes = 524288
```

---

## Protocol & Versioning

### Response envelope versioning

The response envelope includes a `version` field for forward compatibility. Clients should check this to handle schema changes gracefully.

```json
{
  "version": "1",
  "ok": true,
  "tool": "grep",
  "data": { ... },
  "meta": { ... },
  "error": null
}
```

### MCP spec compatibility

osmcp targets **MCP spec 2025-03-26** (the first stable release). Compatibility with newer spec versions will be tracked in the changelog.

### Deprecation policy

- Tool argument fields will not be removed without a **minor version** bump and a **one release** deprecation window.
- Tool names will not be renamed — new names are added alongside old ones, old names removed only in a **major version**.

---

## Open Questions (Expanded)

| Question | Options | Lean |
|----------|---------|------|
| Policy Engine shape | Declarative TOML vs. programmable plugin/rules engine | Start with TOML; plugin API in v2 |
| Policy scope conflict resolution | Project policy restricts only vs. can expand | Project can only restrict |
| Tool visibility model | Capability-based (only granted tools visible) vs. runtime denial | Capability-based (stronger) |
| `run_script` sandbox depth | Path jail only vs. Linux namespaces/landlock | Path jail in v1; namespaces in v2 |
| Network access in `run_script` | Always blocked vs. policy-configurable | Blocked by default, configurable |
| Binary blocklist in `run_script` | Hard-coded vs. fully configurable | Hard default set, operator-extensible |
| Multiple simultaneous agents | Not supported vs. HTTP transport with named profiles | Not in v1; v2 HTTP mode |
| Streaming output | Buffered only vs. MCP streaming protocol | Buffered in v1; streaming in v2 |
| Human-in-the-loop confirmation | Not supported vs. terminal prompt hook | Not in v1; opt-in in v2 |
| `chmod`/`chown` in v1 | Ship disabled vs. defer entirely | Defer to v2 |
| Windows support | Not planned vs. best-effort | Not in v1 |
| Hot-reload policy config | Restart required vs. `SIGHUP` triggers reload | Restart in v1; SIGHUP in v2 |
| Audit log for `run_script` output | Log full output vs. metadata only | Metadata only (size concerns) |

---

## Roadmap

### v1 — Core read + sandbox foundation *(in progress)*
**File Inspection:** `ls`, `cat`, `stat`, `wc`, `head`, `tail`, `tree`, `du`
**Search & Discovery:** `grep`, `find`
**Git Intelligence (read-only):** `git_status`, `git_diff`, `git_log`
**Data Transformation:** `jq`, `sed`, `diff`
**Infrastructure:** Policy Engine (TOML, path jail, tool allowlist, capability visibility), Uniform JSON envelope, Structured audit log (NDJSON), `--validate` flag for CI policy linting, Homebrew formula + static binaries (macOS/Linux arm64/amd64)

### v2 — Mutating ops + surgical edits + streaming
**Mutating Filesystem:** `cp`, `mv`, `rm`, `mkdir`
**Git Write Ops:** `git_add`, `git_commit`, `git_branch`, `git_stash`
**Surgical File Edit (`patch`):** Modify files by line range (insert/replace/delete). Safer than AI rewriting entire files. Gated by `allow_mutation = true`.
**File Watching (`watch`):** Emit MCP notifications when files in a directory change. Allows agents to react to events rather than polling.
**Archiving (`archive`):** Create and extract `.tar.gz`/`.zip` in pure Go.
**Dry-run mode** for all mutating tools
**Project-level `.osmcp/policy.toml`** (committed to repo for team-wide rules)
**HTTP/SSE transport** with named agent profiles and Bearer token auth
**Streaming output** for `run_script` + cancellation via MCP `tools/cancel`
**Human-in-the-loop confirmation hook** for destructive ops
Prometheus metrics endpoint + `GET /healthz` health check
Hot-reload policy on `SIGHUP`

### v3 — Deep sandboxing + ecosystem
- Linux namespaces / landlock for `run_script` (kernel-level network block)
- `seccomp` profile for subprocess syscall restriction
- Docker image (official)
- Rust execution layer rewrite (optional, for deeper OS sandboxing)
- Plugin API for custom typed tools
- IDE extension integrations (VS Code, JetBrains)