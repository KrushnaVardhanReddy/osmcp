# Task 24: Phase 3 & 4 End-to-End Testing

## Goal
Implement End-to-End (E2E) tests for all tools developed in Phase 3 and Phase 4 to ensure they correctly interact with the MCP JSON-RPC stdio layer and the Policy Engine.

## Implementation Details & Context
1. **Test Environment**: Use the `setupMCPClient` helper function established in `e2e/helpers_test.go`.
2. **Tools to Cover**:
   - `run_script` (Task 16)
   - `get_env` (Task 21)
   - `hash_file` (Task 22)
3. **Policy Engine Enforcement**: 
   - Verify that `run_script` successfully executes when `allow_run_script = true`.
   - Verify that `run_script` returns `POLICY_DENIED` when `allow_run_script = false`.
   - Verify that `get_env` correctly reads environment variables mapped in the `env_allowlist`.
   - Verify that `hash_file` calculates the correct SHA-256 for files inside the `allowed_root`, but returns `POLICY_DENIED` for paths outside of it.

## Location
- `e2e/run_script_e2e_test.go`
- `e2e/system_e2e_test.go` (for get_env and hash_file)

## Testing
- Ensure `go test -race ./e2e/...` passes.
