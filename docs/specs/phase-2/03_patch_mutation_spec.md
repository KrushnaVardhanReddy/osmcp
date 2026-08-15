# Phase 2: Patch Mutation Tool

## Overview
This document specifies the behavior of the `patch` mutation tool.
The `patch` tool applies a unified diff to a target file.
It must strictly adhere to the `policy.Engine`'s path and mutation validation rules, returning `-32603` if `allow_mutation` is false.

## Tool Specifications

### 1. `patch`
Applies a unified diff patch to a file.

**Parameters:**
- `path` (string, required): The absolute or relative path to the file to be patched.
- `diff` (string, required): The unified diff text to apply.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(path)`.
- Must check `policyEngine.CanMutate()`.

**Returns:**
- Success: A confirmation message indicating the file was successfully patched.
- Error: Standard envelope error if the patch failed to apply, or if policy violations occurred.

## JSON-RPC MCP Registration
The tool must be self-contained and implement the `RegisterMCP(s *server.MCPServer)` interface.
