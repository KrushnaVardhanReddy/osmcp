# Task 11: File System Mutation Tools

## Goal
Implement the File System Mutation tools specified in `docs/specs/phase-2/01_fs_mutation_spec.md`.

## Contract
Implement the `FSMutationTool` interface defined in `docs/contracts/phase-2/fs_mutation_contract.go`.

## Tools to Implement
- `write_file`
- `append_file`
- `mkdir`
- `rm`
- `mv`
- `cp`

## Implementation Details & Context
1. **Mutation Policy Check (CRITICAL)**: Because these tools mutate the file system, you **MUST** call `policyEngine.CanMutate()` before performing any operations. If it returns false, you must return an error envelope with `-32603`.
2. **Path Validation**: Just like Phase 1, you must call `policyEngine.ValidatePath(path)` for all paths (both source and destination).
3. **Pure Go Implementation**: Use the standard library (`os`, `io`, `path/filepath`). Do not use `exec.Command("cp")` or `exec.Command("rm")`.
4. **Tool Registry**: Ensure each tool struct implements `RegisterMCP(s *server.MCPServer)` so it can register itself with its own JSON schema, exactly as done in Phase 1 tools.
5. **Location**: Place your implementations in `internal/tools/fs_mutation.go` (or individual files like `write_file.go`, `rm.go`, etc.).

## Testing
Write unit tests for your tools in `internal/tools/fs_mutation_test.go` utilizing a mock policy engine that simulates `allow_mutation = true` and `allow_mutation = false`.
