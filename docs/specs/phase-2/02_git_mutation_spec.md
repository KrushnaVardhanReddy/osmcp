# Phase 2: Git Mutation Tools

## Overview
This document specifies the behavior of Git mutation tools: `git_add`, `git_commit`, `git_checkout`, `git_branch`, `git_pull`, and `git_push`.
These tools execute operations on a Git repository using the `github.com/go-git/go-git/v5` library (or native OS commands securely wrapped if `go-git` lacks coverage).
All tools must strictly adhere to the `policy.Engine`'s path and mutation validation rules, returning `-32603` if `allow_mutation` is false.

## Tool Specifications

### 1. `git_add`
Stages changes for the next commit.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `paths` (array of strings, required): Specific paths to stage, or `["."]` for all changes.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

### 2. `git_commit`
Creates a new commit containing the currently staged changes.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `message` (string, required): The commit message.
- `author_name` (string, optional): Author name.
- `author_email` (string, optional): Author email.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

### 3. `git_checkout`
Switches to a different branch or restores files.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `branch` (string, optional): The name of the branch to checkout.
- `create` (boolean, optional, default: false): If true, creates a new branch (`git checkout -b`).

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

### 4. `git_branch`
Creates, lists, or deletes branches.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `action` (string, required): One of `"create"`, `"delete"`.
- `branch_name` (string, required): The name of the branch.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

### 5. `git_pull`
Fetches from and integrates with another repository or a local branch.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `remote` (string, optional, default: "origin"): The remote name.
- `branch` (string, optional): The branch to pull.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

### 6. `git_push`
Updates remote refs along with associated objects.

**Parameters:**
- `repo_path` (string, required): The root path of the Git repository.
- `remote` (string, optional, default: "origin"): The remote name.
- `branch` (string, optional): The branch to push.
- `force` (boolean, optional, default: false): If true, force push.

**Policy Enforcement:**
- Must check `policyEngine.ValidatePath(repo_path)`.
- Must check `policyEngine.Evaluate()`.

## JSON-RPC MCP Registration
Each tool must be self-contained and implement the `RegisterMCP(s *server.MCPServer)` interface.
