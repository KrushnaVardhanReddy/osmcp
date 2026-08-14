# Task 2: grep Tool + MCP Server Entry Point

## Objective
Implement the `grep` typed tool — osmcp's first end-to-end tool — and wire it into a
working MCP server binary. When this task is done, a real MCP client (e.g. Claude
Desktop) can connect to osmcp over stdio, call `grep`, and receive a uniform JSON
envelope response.

> ⚠️ **Spec-first rule**: Do NOT write any implementation code before reading the specs
> and contracts listed in Prerequisites. The contract defines the exact types — your job
> is to implement them, not design them.

## Prerequisites
- Task 1 (Foundation) must be **fully complete and merged** before starting this task.
- Read `docs/specs/phase-1/01_grep_tool_spec.md` fully.
- Read `docs/contracts/phase-1/grep_contract.go` — implement `GrepArgs` and `GrepData`
  exactly as defined. Do not add, rename, or remove fields.
- Read `docs/contracts/cross_cutting/core_contracts.go` — implement the `Tool` interface.

## Implementation Steps

### 1. Implement the grep tool

File: `internal/tools/grep.go`

- Implement the `Tool` interface: `Name()`, `Description()`, `IsMutating()`.
- `Name()` must return exactly `"grep"`.
- `IsMutating()` must return `false`.
- Input struct: use `GrepArgs` from the contract exactly. Do not modify the struct.
- Output struct: use `GrepData` / `GrepMatch` from the contract exactly.

**Execution pipeline (must follow this order):**
1. Validate args: reject empty `pattern` or `path` with `INVALID_ARGS`.
2. Call `PolicyEngine.Evaluate("grep", []string{args.Path}, false)`.
   On error → return `POLICY_DENIED` envelope.
3. Check path exists: `os.Stat(args.Path)`. If missing → `NOT_FOUND`.
4. Configure `grep-go` options based on `GrepArgs` (see spec §4).
   - Set `MaxMatches = min(args.MaxMatches, policy.Limits().MaxMatches)` (0 means use policy limit).
5. Call the `grep-go` library to execute the search in pure Go (no `os/exec`).
6. Translate the library's match results into `[]GrepMatch`.
7. Return `EnvelopeBuilder.Success(...)` or `EnvelopeBuilder.Failure(...)`.

**Truncation:** Stop parsing at `MaxMatches`. Set `meta.Truncated = true`.

**No-match is not an error:** Return `GrepData{Matches: [], Count: 0}` with `ok: true`.

### 2. Implement the tool registry

File: `internal/tools/registry.go`

- Implement `ToolRegistry` interface from `core_contracts.go`.
- `Register(tool Tool)` — stores the tool by name.
- `VisibleTools()` — returns only tools whose `Name()` is in `PolicyEngine.IsToolVisible()`.
- The registry is the only place tools are registered with the MCP server.

### 3. Implement the MCP server entry point

File: `cmd/osmcp/main.go`

```
1. Parse CLI flags: --policy, --audit-log, --validate, --version
2. If --version: print version string and exit 0
3. Load policy via policy.LoadFromFile(). On error: print and exit 1
4. If --validate: run policy.Validate(), print report, exit 0/1
5. Initialize AuditLogger
6. Initialize PolicyEngine with loaded policy and AuditLogger
7. Initialize Executor
8. Initialize EnvelopeBuilder
9. Build ToolRegistry; register GrepTool
10. Create MCP server:
    server := mcp.NewServer(&mcp.Implementation{Name: "osmcp", Version: "0.1.0"}, nil)
11. For each tool in registry.VisibleTools(): register with MCP server
12. server.Run(ctx, &mcp.StdioTransport{})
```

### 4. Write grep unit tests

File: `internal/tools/grep_test.go`

Must cover all 10 invariants from `docs/contracts/phase-1/grep_contract.go`:

- INV-GREP-01: No matches → ok=true, empty array
- INV-GREP-02: Truncated → meta.Truncated=true
- INV-GREP-03: Path outside AllowedRoot → POLICY_DENIED
- INV-GREP-04: grep not in AllowedTools → POLICY_DENIED
- INV-GREP-05: Non-existent path → NOT_FOUND
- INV-GREP-06: Empty pattern → INVALID_ARGS
- INV-GREP-07: Timeout exceeded → TIMEOUT
- INV-GREP-08: Shell injection in pattern → safe (test that `; rm -rf /tmp/test` in pattern does not execute)
- INV-GREP-09: context_lines=2 → correct context populated
- INV-GREP-10: literal=true → dot is not treated as regex wildcard

Use `testdata/fixtures/` for reproducible file content. Create any fixture files needed.

### 5. Write E2E tests

File: `e2e/grep_e2e_test.go`

Helper: `e2e/helpers_test.go` — starts a real `osmcp` process, connects via stdio,
sends MCP JSON-RPC messages, returns parsed responses.

E2E test cases:
- Full round-trip: `tools/list` → grep is visible → call grep → assert envelope shape
- Policy denial: path outside root → assert POLICY_DENIED error envelope
- tools/list with grep excluded from policy → grep is NOT in the response

Use `testdata/policies/readonly.toml` from Task 1.

### 6. Add Makefile targets

```makefile
.PHONY: build test lint e2e validate-policy

build:
	go build -o bin/osmcp ./cmd/osmcp

test:
	go test -race ./internal/...

e2e: build
	go test -race ./e2e/...

lint:
	go vet ./...

validate-policy:
	./bin/osmcp --validate --policy .osmcp/policy.toml
```

## Definition of Done
- `make build` succeeds → `bin/osmcp` binary exists.
- `make test` passes with `-race` flag — zero data races.
- `make e2e` passes — real MCP round-trip works over stdio.
- `make lint` passes clean.
- Connecting Claude Desktop (or any MCP client) to osmcp with a readonly policy:
  - `tools/list` returns `grep` with its full JSON schema.
  - Calling grep on a file within allowed_root returns a correct envelope.
  - Calling grep on `/etc/passwd` returns `POLICY_DENIED`.
- **No mocks** in tests. Real policy engine, real grep binary, real MCP transport.
- All 10 `INV-GREP-*` invariants have passing unit tests.

## Files to Create
```
internal/tools/grep.go
internal/tools/grep_test.go
internal/tools/registry.go
cmd/osmcp/main.go
e2e/helpers_test.go
e2e/grep_e2e_test.go
testdata/fixtures/sample.go       (a Go file with TODO comments for grep tests)
testdata/fixtures/sample.txt      (plain text file for literal search tests)
testdata/fixtures/nested/deep.go  (for recursive search tests)
.osmcp/policy.toml                (osmcp's own policy — dogfooding)
Makefile
```
