# Anthropic MCP Servers Registry Submission

**Repository:** `KrushnaVardhanReddy/osmcp`
**Target Registry:** [https://github.com/modelcontextprotocol/servers](https://github.com/modelcontextprotocol/servers)

## Pull Request Details

### Title
`Add osmcp (OS Capabilities for AI Agents) to the registry`

### Description
```markdown
This PR adds `osmcp` to the official MCP server registry.

`osmcp` provides a typed, policy-controlled OS capability layer for AI agents. It exposes a curated set of safe filesystem, git, and text-processing tools, all governed by a strict Policy Engine (TOML) that enforces path boundaries, tool allowlists, output limits, mutation controls, and an immutable audit trail.

**Capabilities:**
- File Inspection & Transformation (grep, find, ls, cat, jq, sed, awk)
- Filesystem Mutation (write, append, mkdir, rm, mv, cp)
- Git Intelligence & Mutation (status, diff, log, commit, push, pull, patch)
- Secure execution environments with environment variable whitelists and cryptographic hashing.

Built entirely in Go for speed, low memory overhead, and single-binary deployment.
```

## JSON Payload for `mcpservers.json`

Add this block to the appropriate category (e.g., `Developer Tools` or `System`):

```json
{
  "name": "osmcp",
  "description": "A typed, policy-controlled OS capability layer for AI agents. Provides safe, bounded filesystem access, git intelligence, text transformation, and process sandboxing via a configurable TOML policy.",
  "vendor": "KrushnaVardhanReddy",
  "sourceUrl": "https://github.com/KrushnaVardhanReddy/osmcp",
  "command": "osmcp",
  "args": ["--policy", "policy.toml"]
}
```
