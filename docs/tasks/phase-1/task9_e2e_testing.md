# Task 9: Phase 1 End-to-End Testing

## Objective
Validate all 15 Phase 1 features (Policy Engine, server entry point, and ALL Phase 1 tools) via End-to-End (E2E) testing by interacting with the compiled `osmcp` binary over stdio using MCP JSON-RPC.

## Prerequisites
- Tasks 1 through 8 are already fully completed and merged.
- The `osmcp` binary architecture has been refactored to use Self-Registering tools (`RegisterMCP(s *server.MCPServer)`). 
- **CRITICAL**: Do NOT modify `cmd/osmcp/main.go`, `core_contracts.go`, or any of the `internal/tools/*.go` files! Your task is strictly limited to writing tests in the `e2e/` package.

## Implementation Steps

### 1. E2E Test Setup (Already Partially Complete)
- The test harness is already built. Review `e2e/helpers_test.go` and `e2e/grep_e2e_test.go` to understand how the `osmcp` binary is started as a subprocess and how MCP JSON-RPC requests are sent.
- **IMPORTANT**: When checking tool responses, you must unmarshal the response string into `contracts.Envelope`, and then convert `env.Data` into the correct structs (e.g., `contracts_phase1.LsData`). See `TestE2E_Grep_Success` for the exact pattern.

### 2. E2E Test Cases for All Tools (Split by Domain)
Since there are 14 new tools, **DO NOT put all tests in a single file.** To avoid context limits and maintain organization, create the following separate test files:
- `e2e/file_inspection_e2e_test.go` (ls, cat, stat, wc)
- `e2e/git_tools_e2e_test.go` (git_status, git_diff, git_log) *Note: Initialize a temporary git repository using `os/exec` for these tests.*
- `e2e/find_e2e_test.go` (find)
- `e2e/transform_e2e_test.go` (jq, sed, diff)
- `e2e/utility_e2e_test.go` (tree, head, tail, du)

**For each tool, implement:**
1. **Happy Path:** Send a `CallTool` request targeting known fixtures in `testdata/fixtures`. Assert `ok=true` and correct behavior.
2. **Policy Denial:** Send a request with a `path` explicitly outside the `allowed_root` (e.g., `/etc/passwd`). Assert `ok=false` and `env.Error.Code == contracts.ErrPolicyDenied`.

### 3. Tool Visibility Tests
- Create `e2e/visibility_e2e_test.go` to test the `tools/list` endpoint.
- Write a test using a policy where ALL 15 tools are enabled. Assert they are all present in the `res.Tools` slice.
- Write a test using a policy where only half the tools are in `allowed_tools`. Assert that only the allowed tools are visible, and the omitted tools are completely hidden.

## Definition of Done
- All E2E tests pass reliably without flakiness (`go test -race ./e2e/...`).
- Tests use the real compiled binary and test real MCP JSON-RPC communication over stdio.
- **NO MOCKS**: Because this is E2E testing using a compiled binary, do not create `mockPolicyEngine` or any other mock structs. The binary uses the real Policy Engine.
