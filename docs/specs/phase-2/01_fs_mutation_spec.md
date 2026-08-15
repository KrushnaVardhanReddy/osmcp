# Phase 2: File System Mutation Tools

## Overview
This document specifies the behavior of core file system mutation tools: `write_file`, `append_file`, `mkdir`, `rm`, `mv`, and `cp`.
These tools must strictly adhere to the `policy.Engine`'s path and mutation validation rules. Specifically, they must return `-32603` (Policy Violation) if the server's `allow_mutation` flag is false.

## Tool Specifications

### 1. `write_file`
Creates a new file or overwrites an existing file with the provided content.

**Parameters:**
- `path` (string, required): The absolute or relative path to the file.
- `content` (string, required): The content to write to the file.
- `overwrite` (boolean, optional, default: false): If false, the tool must return an error if the file already exists.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(path)`.
- Must check `policyEngine.CanMutate()`.

**Returns:**
- Success: Confirmation message with the number of bytes written.
- Error: Standard envelope error if policy violation or IO failure.

### 2. `append_file`
Appends content to an existing file. If the file does not exist, it creates it.

**Parameters:**
- `path` (string, required): The absolute or relative path to the file.
- `content` (string, required): The content to append.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(path)`.
- Must check `policyEngine.CanMutate()`.

### 3. `mkdir`
Creates a directory. It must automatically create parent directories if they do not exist (equivalent to `mkdir -p`).

**Parameters:**
- `path` (string, required): The absolute or relative path of the directory.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(path)`.
- Must check `policyEngine.CanMutate()`.

### 4. `rm`
Removes a file or directory.

**Parameters:**
- `path` (string, required): The absolute or relative path to remove.
- `recursive` (boolean, optional, default: false): Must be true to remove directories. If false and the target is a directory, it must return an error.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(path)`.
- Must check `policyEngine.CanMutate()`.
- *Safety Rule*: Must explicitly reject an attempt to `rm` the policy `allowed_root` directory itself.

### 5. `mv`
Moves or renames a file or directory.

**Parameters:**
- `source` (string, required): The path of the source item.
- `destination` (string, required): The path of the destination.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(source)` AND `policyEngine.ValidatePath(destination)`.
- Must check `policyEngine.CanMutate()`.

### 6. `cp`
Copies a file or directory recursively.

**Parameters:**
- `source` (string, required): The path of the source item.
- `destination` (string, required): The path of the destination.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(source)` AND `policyEngine.ValidatePath(destination)`.
- Must check `policyEngine.CanMutate()`.

## JSON-RPC MCP Registration
Each tool must be self-contained and implement the `RegisterMCP(s *server.MCPServer)` interface to bind its JSON schema and handler.
