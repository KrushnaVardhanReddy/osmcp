# Task 1: Go Module Bootstrap + Core Abstractions

## Objective
Initialize the osmcp Go module, wire up the official MCP SDK, and implement the three
cross-cutting foundation packages: **response envelope**, **policy engine**, and
**execution engine**. No tool implementations yet — only the shared infrastructure that
every tool will build on.

> ⚠️ **Spec-first rule**: Do NOT write any implementation code before reading the specs
> and contracts listed in Prerequisites. The contracts define the exact interfaces — your
> job is to implement them, not design them.

## Prerequisites
- Read `docs/specs/cross_cutting/01_response_envelope_spec.md` fully before writing any code.
- Read `docs/specs/cross_cutting/02_policy_engine_spec.md` fully before writing any code.
- Read `docs/contracts/cross_cutting/core_contracts.go` — this is the interface contract
  you must implement exactly. Do not rename interfaces or change signatures.

## Implementation Steps

### 1. Initialize Go module

```bash
go mod init github.com/osmcp/osmcp
go get github.com/modelcontextprotocol/go-sdk
go get github.com/BurntSushi/toml
go get github.com/stretchr/testify
```

Create `go.mod` requiring Go 1.22+. Commit `go.mod` and `go.sum`.

### 2. Implement `internal/response` package

File: `internal/response/envelope.go`

Implement the `EnvelopeBuilder` interface from `core_contracts.go`:
- `Success(tool, data, meta) Envelope`
- `Failure(tool, code, message, retryable, meta) Envelope`
- Version field must always be `"1"`.
- `data` must be `null` (not omitted) on failure.
- `error` must be `null` (not omitted) on success.

### 3. Implement `internal/policy` package

File: `internal/policy/policy.go`
- Define the `Policy` struct matching the TOML schema in spec §2 (including `ReadOnlyPaths`).
- Implement `LoadFromFile(path string) (*Policy, error)` using BurntSushi/toml.
- Implement `Validate(*Policy) []error` — used by `--validate` CLI flag.
  Validation rules: spec §6.
- Implement `DefaultPolicy() *Policy` — safe defaults (read-only, cwd as root).

File: `internal/policy/engine.go`
- Implement `PolicyEngine` interface from `core_contracts.go`.
- `Evaluate()` must follow the exact algorithm in spec §3 — check order matters.
- Symlink resolution: use `filepath.EvalSymlinks` before prefix comparison.
- `Evaluate()` must enforce `ReadOnlyPaths` if `isMutating` is true.
- `IsToolVisible()` must check `policy.AllowedTools` (case-sensitive exact match).
- `Limits()` returns the `PolicyLimits` from the loaded policy.



### 4. Implement `internal/audit` package

File: `internal/audit/logger.go`
- Implement `AuditLogger` interface from `core_contracts.go`.
- Write one NDJSON line per `Log(AuditEntry)` call.
- Support destinations: `"stderr"` (default) and `"file"` (path from config).
- Use `sync.Mutex` for goroutine-safe writes.
- Timestamps must be RFC3339 with millisecond precision.

## Definition of Done
- `go build ./...` succeeds with zero errors.
- `go vet ./...` passes clean.
- Unit tests pass:
  - `internal/response`: Success and Failure envelopes serialize to exactly the shapes in spec §5 and §6.
  - `internal/policy`: All 12 evaluation cases in spec §3 produce correct ALLOW/DENY results.
  - `internal/policy`: `Validate()` catches invalid `allowed_root`, unknown tool names, out-of-range limits.

  - `internal/audit`: NDJSON output is valid JSON-per-line; mutex prevents data races under `-race`.
- **No mock implementations.** All tests use real types, not stubs.
- **No tool implementations yet.** This task ends at the foundation layer.

## Files to Create
```
go.mod
go.sum
internal/response/envelope.go
internal/response/envelope_test.go
internal/policy/policy.go
internal/policy/engine.go
internal/policy/engine_test.go

internal/audit/logger.go
internal/audit/logger_test.go
testdata/policies/readonly.toml
testdata/policies/mutation_allowed.toml
testdata/policies/empty.toml
```
