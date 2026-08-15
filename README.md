# osmcp — OS Capabilities for AI Agents

> **A typed, policy-controlled OS capability layer for AI agents via the Model Context Protocol (MCP).**

osmcp exposes a curated set of read-only filesystem, git, and text-processing tools to AI agents — all governed by a strict **Policy Engine** that enforces path boundaries, tool allowlists, output limits, and an immutable audit trail.

## Features (Phase 1 — v1.0.0)

| Category | Tools |
|---|---|
| 🔍 **Search** | `grep`, `find` |
| 📁 **File Inspection** | `ls`, `cat`, `stat`, `wc`, `head`, `tail` |
| 🌳 **Filesystem** | `tree`, `du` |
| 🔀 **Git Intelligence** | `git_status`, `git_diff`, `git_log` |
| 🔧 **Transform** | `jq`, `sed`, `diff` |

## Architecture

```
AI Agent (Claude, GPT, etc.)
    │  MCP JSON-RPC (stdio)
    ▼
osmcp binary
    ├── Policy Engine      ← enforces allowed_root, allowed_tools, limits
    ├── Audit Logger       ← append-only NDJSON log of every invocation
    ├── Tool Registry      ← self-registering tools via RegisterMCP()
    └── Envelope Builder   ← typed {ok, data, error, meta} responses
```

## Quick Start

### 1. Build

```bash
make build
# Binary: bin/osmcp
```

### 2. Configure a Policy

```toml
# policy.toml
[policy]
allowed_root   = "/home/user/myproject"
allowed_tools  = ["grep", "ls", "cat", "git_status", "git_log"]
allow_mutation = false

[limits]
timeout_ms       = 5000
max_output_bytes = 1048576
max_matches      = 100

[audit]
destination = "stderr"   # or "file"
path        = "/var/log/osmcp-audit.ndjson"
```

### 3. Run

```bash
bin/osmcp --policy policy.toml
```

The binary communicates over **stdio** using MCP JSON-RPC. Connect any MCP-compatible client (Claude Desktop, Cursor, etc.).

### 4. Test

```bash
make test     # unit tests
make e2e      # end-to-end tests against real binary
make lint     # golangci-lint
```

## Policy Security Model

- **`allowed_root`** — All filesystem paths are validated to be inside this root. Traversal outside is blocked with `POLICY_DENIED`.
- **`allowed_tools`** — Only tools in this list are visible to the MCP client. Unlisted tools do not appear in `tools/list`.
- **`allow_mutation`** — When `false`, mutating tools (write, delete, git commit) are globally blocked.
- **Limits** — Per-invocation timeout, output byte cap, and match count cap prevent runaway operations.

## Envelope Response Format

All tool responses follow a consistent typed envelope:

```json
{
  "ok": true,
  "tool": "grep",
  "data": { ... },
  "error": null,
  "meta": {
    "execution_time_ms": 12,
    "truncated": false
  }
}
```

## License

MIT
