# Task 9: Phase 1 End-to-End Testing

## Objective
Validate all Phase 1 features (Policy Engine, server entry point, and ALL Phase 1 tools) via End-to-End (E2E) testing by interacting with the compiled `osmcp` binary over stdio using MCP JSON-RPC.

## Prerequisites
- Tasks 1 through 8 must be fully complete and merged.
- Real `osmcp` binary must be compiled.

## Implementation Steps

### 1. E2E Test Setup
- Ensure `e2e/helpers_test.go` can reliably start the `osmcp` binary as a subprocess with `stdin` and `stdout` pipes.
- Implement helper functions to send JSON-RPC requests (e.g., `Initialize`, `CallTool`) and parse JSON-RPC responses.

### 2. E2E Test Cases for All Tools
- Implement `e2e/tools_e2e_test.go` to test each tool:
  - **Happy Paths:** Send a `CallTool` request for `grep`, `ls`, `cat`, `git_status`, `find`, `jq`, `tree`, etc. targeting known fixtures. Assert `ok=true` and correct matches/behavior.
  - **Policy Denial:** Send a request with a `path` outside `allowed_root` for any filesystem tool. Assert `ok=false` and `code=POLICY_DENIED`.
  - **Tool Visibility:** Send `tools/list` with a policy where all tools are enabled. Assert they are all present.
  - **Tool Hidden:** Send `tools/list` with a policy where specific tools are removed from `allowed_tools`. Assert they are NOT present.

### 3. CI Integration
- Verify that `make e2e` runs these tests as part of the GitHub Actions CI pipeline (`.github/workflows/ci.yml`).

## Definition of Done
- All E2E tests pass reliably without flakiness.
- Tests use the real compiled binary and test real MCP JSON-RPC communication over stdio.
- No mocks are used for the server component.
