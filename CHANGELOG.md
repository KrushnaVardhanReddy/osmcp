# Changelog

All notable changes to osmcp are documented here.
Follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and [Semantic Versioning](https://semver.org/).

---

## [v1.0.0] — 2026-08-15

### 🎉 Phase 1 — Initial Release

This is the first stable release of **osmcp**, a typed, policy-controlled OS capability layer for AI agents.

### Added

#### Core Infrastructure (Task 1)
- **Policy Engine** — enforces `allowed_root`, `allowed_tools`, `allow_mutation`, and per-request `limits`
- **Audit Logger** — append-only NDJSON log of every tool invocation (stderr or file destination)
- **Envelope Builder** — typed `{ok, data, error, meta}` response format for all tools
- **Tool Registry** — self-registering MCP tools via `RegisterMCP(s *server.MCPServer)` interface

#### Search Tools (Task 2 & Task 6)
- **`grep`** — Search for patterns in files and directories with recursive support, context lines, case-insensitive mode, and match count truncation
- **`find`** — Traverse directories with filters for name glob, type (file/dir), size range, modification time, and max depth

#### File Inspection Tools (Task 4)
- **`ls`** — List directory contents with symlink resolution and metadata
- **`cat`** — Read file contents with streaming and byte-limit truncation
- **`stat`** — Return file metadata (size, permissions, timestamps, type)
- **`wc`** — Count lines, words, and bytes in files

#### Git Intelligence Tools (Task 5)
- **`git_status`** — Show working tree status (staged, unstaged, untracked)
- **`git_diff`** — Show unified diff between commits or working tree
- **`git_log`** — Return structured commit history with author, timestamp, message

#### Transform Tools (Task 7)
- **`jq`** — Query and filter JSON using jq expressions (Pure Go via `itchyny/gojq`)
- **`sed`** — Apply `s/pattern/replacement/flags` substitutions (Pure Go regex)
- **`diff`** — Produce unified diffs between two text strings

#### Utility Tools (Task 8)
- **`tree`** — Render directory tree with depth, hidden file, and dirs-only filters
- **`head`** — Return the first N lines of a file
- **`tail`** — Return the last N lines of a file
- **`du`** — Summarize disk usage by subdirectory

#### End-to-End Testing (Task 9)
- 36 E2E tests covering all 15 tools over real MCP JSON-RPC stdio communication
- Tests validate happy paths, policy denial, and tool visibility
- Zero mocks — all tests run against the compiled `bin/osmcp` binary

### Architecture

- **100% Pure Go** — No shell exec, no cgo. All tools use the Go standard library (`os`, `io`, `path/filepath`) for platform independence and security
- **Spec-first** — All tools implemented from contracts in `docs/contracts/phase-1/`
- **Self-registering** — Each tool owns its own MCP schema via `RegisterMCP()`, eliminating merge conflicts during parallel development

[v1.0.0]: https://github.com/KrushnaVardhanReddy/osmcp/releases/tag/v1.0.0
