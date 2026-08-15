# `hash_file` Specification

## Overview
`hash_file` is a read-only tool that calculates the cryptographic hash of a file on the host filesystem. This allows an AI agent to verify file integrity, deduplicate files, or check if a file has changed without needing to read its entire contents via the MCP protocol.

## Features
- Supports common algorithms: `sha256` (default), `md5`, `sha1`.
- Fast path for large files: Streams the file contents directly into the hash function to maintain a constant, low memory footprint.

## Security Constraints
- Follows the standard `allowed_root` boundary checks via the Policy Engine. Returns `POLICY_DENIED` if attempting to hash a file outside the workspace.
- Enforces `timeout_ms` limit strictly, failing gracefully if a massive file takes too long to hash.

## Output
Returns the string representation of the computed hash and the algorithm used.
