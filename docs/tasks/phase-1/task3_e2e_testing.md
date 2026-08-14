# Task 3: Phase 1 End-to-End Testing

## Objective
Validate all Phase 1 features (Policy Engine, Executor, grep tool, envelope generation) via End-to-End (E2E) testing by interacting with the compiled `osmcp` binary over stdio using MCP JSON-RPC.

## Prerequisites
- Task 1 and Task 2 must be fully complete and merged.
- Real `osmcp` binary must be compiled.

## Implementation Steps

### 1. E2E Test Setup
- Ensure `e2e/helpers_test.go` can reliably start the `osmcp` binary as a subprocess with `stdin` and `stdout` pipes.
- Implement helper functions to send JSON-RPC requests (e.g., `Initialize`, `CallTool`) and parse JSON-RPC responses.

### 2. E2E Test Cases for `grep`
- Implement `e2e/grep_e2e_test.go`:
  - **Happy Path:** Send a `CallTool` request for `grep` targeting a known fixture in `testdata/fixtures/`. Assert `ok=true` and correct matches.
  - **Policy Denial:** Send a request with a `path` outside `allowed_root`. Assert `ok=false` and `code=POLICY_DENIED`.
  - **Tool Visibility:** Send `tools/list` with a policy where `grep` is enabled. Assert `grep` is present.
  - **Tool Hidden:** Send `tools/list` with a policy where `grep` is removed from `allowed_tools`. Assert `grep` is NOT present.

### 3. CI Integration
- Verify that `make e2e` runs these tests as part of the GitHub Actions CI pipeline (`.github/workflows/ci.yml`).

## Definition of Done
- All E2E tests pass reliably without flakiness.
- Tests use the real compiled binary and test real MCP JSON-RPC communication over stdio.
- No mocks are used for the server component.
