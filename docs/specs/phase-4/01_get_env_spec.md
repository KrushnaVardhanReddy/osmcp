# `get_env` Specification

## Overview
`get_env` is a diagnostic tool that securely returns environment variables from the host environment to the AI agent.

## Security Constraints
- **Implicit Blocklist by Default:** Never return the full environment to prevent accidental secret leakage (e.g., `AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`).
- **Explicit Allowlist Required:** The policy engine must define `policy.env_allowlist = ["GOOS", "GOARCH", "NODE_ENV"]`. Only variables explicitly present in this list can be returned.
- If `env_allowlist` is empty or missing, `get_env` should return an empty map or an error indicating no variables are whitelisted.

## Output
Returns a JSON object mapping environment variable keys to their values.
