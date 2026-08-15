# Task 25: LiteLLM MCP Ecosystem Publishing

## Goal
Publish `osmcp` as a verified, discoverable MCP server within the LiteLLM ecosystem so that
users who already have a LiteLLM gateway can connect `osmcp` as a managed tool backend with
zero additional configuration.

## Background
LiteLLM (https://github.com/BerriAI/litellm) is an open-source AI Gateway used by 100k+ developers
that now ships a built-in **MCP Gateway**. By listing `osmcp` as a compatible server, we gain access
to their user base — teams who are already managing LLM budgets, API keys, and tool routing through
LiteLLM and want a safe OS capability layer for their agents.

## Research Questions
- [ ] How does LiteLLM's MCP Gateway discover and register external MCP servers?
- [ ] Is there an official MCP server registry or a `mcpservers.json`-style listing?
- [ ] Does LiteLLM have a contribution process for adding community MCP servers (like Smithery)?
- [ ] What config format does LiteLLM use to wire up an external MCP server (stdio vs HTTP)?
- [ ] Are there any compatibility requirements (e.g. specific MCP protocol version)?

## Proposed Actions

### 1. Verify LiteLLM MCP Compatibility
Test `osmcp` against LiteLLM's MCP Gateway:
```yaml
# litellm_config.yaml
mcp_servers:
  osmcp:
    command: osmcp
    args: ["--policy", ".osmcp/policy.toml"]
    transport: stdio
```

### 2. Open a PR to LiteLLM's MCP server list (if they have one)
Similar to how we'd open a PR to Smithery or Anthropic's registry.

### 3. Add a LiteLLM integration guide to our docs
Create `docs/integrations/litellm.md` showing:
- How to wire `osmcp` into a LiteLLM proxy config
- How LiteLLM's key-based access control works with `osmcp`'s policy engine (they layer nicely)
- Example: use LiteLLM team keys to control which agent gets which `osmcp` policy profile

### 4. Add LiteLLM badge to README
Once listed, add an official badge to the README.

## Notes
- LiteLLM's MCP Gateway is relatively new — validate that it supports stdio transport first.
- Our `osmcp --init --profile dev-agent` scaffolding already makes it easy for LiteLLM users to get a policy file.
- LiteLLM's audit logging and our own audit NDJSON log complement each other well — worth documenting.
