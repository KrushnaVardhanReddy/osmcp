# osmcp — OS Capabilities for AI Agents

> **A typed, policy-controlled OS capability layer for AI agents via the Model Context Protocol (MCP).**

osmcp exposes a curated set of safe filesystem, git, and text-processing tools to AI agents — all governed by a strict **Policy Engine** that enforces path boundaries, tool allowlists, output limits, mutation controls, and an immutable audit trail.

📖 **Read the comprehensive [Architecture & Design Document](docs/ARCHITECTURE.md)** for a deep dive into the philosophy, safety boundaries, and design decisions behind osmcp.

## Features

| Category | Tools | Phase |
|---|---|---|
| 🔍 **Search** | `grep`, `find` | 1 |
| 📁 **File Inspection** | `ls`, `cat`, `stat`, `wc`, `head`, `tail` | 1 |
| 🌳 **Filesystem** | `tree`, `du` | 1 |
| 🔀 **Git Intelligence** | `git_status`, `git_diff`, `git_log` | 1 |
| 🔧 **Transform** | `jq`, `sed`, `diff` | 1 |
| ✍️ **File Mutation** | `write_file`, `append_file`, `mkdir`, `rm`, `mv`, `cp`, `patch` | 2 |
| 🚀 **Git Mutation** | `git_add`, `git_commit`, `git_checkout`, `git_branch`, `git_pull`, `git_push` | 2 |

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

### 1. Install via Homebrew

```bash
brew tap KrushnaVardhanReddy/tap
brew install osmcp
```

*Alternatively, build from source:*
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
