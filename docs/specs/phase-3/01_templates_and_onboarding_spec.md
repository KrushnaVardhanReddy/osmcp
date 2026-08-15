# Phase 3: Templates & Agent Onboarding

## Overview

This document specifies the templates and onboarding artifacts that `osmcp` ships with out-of-the-box.
The goal is to reduce the time from "install osmcp" to "productive AI agent" to under 5 minutes, and
to eliminate the trial-and-error phase that agents currently go through when integrating with a new MCP server.

Templates are **not** implemented in Go. They are static files committed under `templates/` in the repository
and installed alongside the binary by GoReleaser. The `osmcp` binary exposes a `--init` flag to scaffold
them into the user's project directory.

---

## 1. Policy Templates

Pre-built `policy.toml` files covering the four most common agent deployment scenarios.

### 1.1 `read-only.toml`
A safe, zero-mutation configuration. Suitable for code review agents, documentation agents,
and any agent that must never modify state.

```toml
[policy]
allowed_root   = "/path/to/project"   # User must fill this in
allowed_tools  = ["grep", "ls", "cat", "stat", "wc", "head", "tail", "tree", "du", "find",
                  "git_status", "git_diff", "git_log", "jq", "sed", "diff"]
allow_mutation = false

[limits]
timeout_ms       = 5000
max_output_bytes = 524288   # 512 KB
max_matches      = 200

[audit]
destination = "stderr"
```

### 1.2 `dev-agent.toml`
Full read + mutation access. Suitable for a trusted local development agent (Claude in Cursor,
Antigravity, etc.) working inside a single project directory.

```toml
[policy]
allowed_root   = "/path/to/project"
allowed_tools  = ["grep", "ls", "cat", "stat", "wc", "head", "tail", "tree", "du", "find",
                  "git_status", "git_diff", "git_log", "jq", "sed", "diff",
                  "write_file", "append_file", "mkdir", "rm", "mv", "cp",
                  "git_add", "git_commit", "git_checkout", "git_branch",
                  "patch"]
allow_mutation = true

[limits]
timeout_ms       = 10000
max_output_bytes = 2097152   # 2 MB
max_matches      = 500

[audit]
destination = "file"
path        = ".osmcp/audit.ndjson"
```

### 1.3 `ci-agent.toml`
Read-only access scoped strictly to the repository root. No mutation, tight output limits.
Suitable for CI/CD pipelines where osmcp is used to give a code analysis agent access
to the checked-out workspace.

```toml
[policy]
allowed_root   = "."
allowed_tools  = ["grep", "find", "ls", "cat", "git_status", "git_diff", "git_log"]
allow_mutation = false

[limits]
timeout_ms       = 3000
max_output_bytes = 262144   # 256 KB
max_matches      = 100

[audit]
destination = "stderr"
```

### 1.4 `review-agent.toml`
Read + git-read only (no file mutation). Git commits, pushes and checkouts are blocked.
Suitable for PR review agents that need to inspect code and git history but must never
commit anything.

```toml
[policy]
allowed_root   = "/path/to/project"
allowed_tools  = ["grep", "ls", "cat", "stat", "wc", "head", "tail", "tree",
                  "git_status", "git_diff", "git_log", "find", "diff", "jq"]
allow_mutation = false

[limits]
timeout_ms       = 5000
max_output_bytes = 1048576   # 1 MB
max_matches      = 300

[audit]
destination = "stderr"
```

---

## 2. System Prompt Templates

Pre-written system prompt snippets that inform an LLM exactly which `osmcp` tools are available,
what they do, and how to call them correctly. Agents that are initialized with these prompts
require zero "tool discovery" exploration — they start productive immediately.

### 2.1 `system-prompts/full-toolset.md`
A complete system prompt listing all Phase 1 + Phase 2 tools with parameter descriptions,
example return shapes, and common error codes. ~500 lines.

**Key sections:**
- Tool table (name, purpose, key parameters)
- Response Envelope format (with a JSON example)
- Policy error handling guide (`POLICY_DENIED`, `NOT_FOUND`, `EXEC_FAILED`)
- Workflow examples: "How to search and edit a file," "How to stage and commit a change"

### 2.2 `system-prompts/read-only.md`
Trimmed version covering only Phase 1 read tools. Suitable for documentation or review agents.

---

## 3. Example Response Schemas

Static JSON files showing the exact response shape for every tool, with both success and error
variants. These allow agents to validate their parsing logic without making a live call.

Location: `templates/examples/`

Files:
- `grep_success.json`, `grep_error.json`
- `ls_success.json`, `ls_error.json`
- `cat_success.json`, `cat_success_truncated.json`, `cat_error.json`
- `git_status_success.json`
- `write_file_success.json`, `write_file_error_policy.json`
- `patch_success.json`, `patch_error_hunk.json`
- *(one success + one error for every tool)*

---

## 4. `osmcp --init` Scaffolding Command

The binary gains a `--init` flag that scaffolds a starter configuration into the current directory.

```
Usage: osmcp --init [--profile read-only|dev-agent|ci-agent|review-agent]
```

**Behavior:**
1. Creates `.osmcp/` directory in CWD.
2. Copies the selected policy template to `.osmcp/policy.toml`.
3. Prints a one-line next-step instruction: `Run: osmcp --policy .osmcp/policy.toml`

**Non-destructive:** If `.osmcp/policy.toml` already exists, the command exits with an error
rather than overwriting it.

---

## Invariants

- **INV-T-01**: All template files are read-only documentation. They contain no generated code.
- **INV-T-02**: All four policy templates must be valid TOML that parses without error.
- **INV-T-03**: All example JSON files must be valid JSON and must deserialize into the `Envelope` struct.
- **INV-T-04**: `osmcp --init` must be idempotent with respect to a missing `.osmcp/` directory (creates it).
- **INV-T-05**: `osmcp --init` must not overwrite an existing `.osmcp/policy.toml`.
- **INV-T-06**: System prompt templates must reference only tools present in the `allowed_tools` list of the matching policy template.
