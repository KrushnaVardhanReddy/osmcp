# Task 13: Patch Mutation Tool

## Goal
Implement the Patch Mutation tool specified in `docs/specs/phase-2/03_patch_mutation_spec.md`.

## Contract
Implement the `PatchMutationTool` interface defined in `docs/contracts/phase-2/patch_mutation_contract.go`.

## Tools to Implement
- `patch`

## Implementation Details & Context
1. **Mutation Policy Check (CRITICAL)**: You **MUST** call `policyEngine.CanMutate()` before attempting to apply the patch. If false, return `-32603`.
2. **Path Validation**: Use `policyEngine.ValidatePath(path)` to verify the target file is within the `allowed_root`.
3. **Patch Library**: You may use a Pure Go library like `github.com/hexops/gotextdiff` or `github.com/sergi/go-diff` to apply the unified diff to the target file.
4. **Tool Registry**: Implement `RegisterMCP(s *server.MCPServer)` so the `patch` tool registers its JSON schema independently.
5. **Location**: Place your implementation in `internal/tools/patch.go`.

## Testing
Write unit tests in `internal/tools/patch_test.go` applying a diff to an in-memory string or temporary file to verify it applies successfully. Ensure `allow_mutation = false` correctly blocks the execution.
