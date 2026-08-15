# Task 14: Phase 2 End-to-End Testing

## Goal
Implement End-to-End (E2E) tests for all Phase 2 Mutation tools, verifying that they correctly interact with the MCP JSON-RPC stdio layer, the Policy Engine, and the underlying filesystem/git repos.

## Implementation Details & Context
1. **Test Environment**: Use the same helper functions established in `e2e/helpers_test.go` during Phase 1.
2. **Mutation Testing**: 
   - Write tests that configure `allow_mutation = true` and verify the files/repos are actually modified.
   - Write tests that configure `allow_mutation = false` and verify that the tools return `-32603` (Policy Violation) and DO NOT modify the filesystem/repo.
3. **Safety**: Since these tools mutate the filesystem, your tests **MUST** create temporary directories (`t.TempDir()`) to serve as the `allowed_root` for the `policy.toml` passed to the `osmcp` binary. DO NOT run mutation tools against the actual project source tree during E2E testing!
4. **Tool Coverage**: Write tests for `write_file`, `append_file`, `mkdir`, `rm`, `mv`, `cp`, `git_add`, `git_commit`, `git_checkout`, `git_branch`, `git_pull`, `git_push`, and `patch`.
