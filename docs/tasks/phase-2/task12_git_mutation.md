# Task 12: Git Mutation Tools

## Goal
Implement the Git Mutation tools specified in `docs/specs/phase-2/02_git_mutation_spec.md`.

## Contract
Implement the `GitMutationTool` interface defined in `docs/contracts/phase-2/git_mutation_contract.go`.

## Tools to Implement
- `git_add`
- `git_commit`
- `git_checkout`
- `git_branch`
- `git_pull`
- `git_push`

## Implementation Details & Context
1. **Mutation Policy Check (CRITICAL)**: You **MUST** call `policyEngine.CanMutate()` before modifying the Git repo. If false, return `-32603`.
2. **Library**: Use `github.com/go-git/go-git/v5` just like we did for Phase 1 Git tools.
3. **Commit Signatures**: For `git_commit`, if the user provides `author_name` and `author_email`, construct an `object.Signature` with the current time. If omitted, try to read the repository config or return an error.
4. **Tool Registry**: Implement `RegisterMCP(s *server.MCPServer)` for each tool.
5. **Location**: Place your implementations in `internal/tools/git_mutation.go` (or individual files).

## Testing
Write unit tests using `go-git/v5/memory` or a temporary filesystem repo to validate the mutation commands without impacting the real repo. Test the policy engine's `CanMutate` boundary as well.
