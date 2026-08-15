# Task 23: Distribution & Launch Materials

## Goal
Prepare the repository for broad adoption by updating the `README.md` with client integrations, creating a demo recording, and preparing the submission for the Anthropic MCP Directory.

## Deliverables

### 1. Claude Desktop Integration
Update `README.md` to include a clear, copy-paste JSON snippet for `claude_desktop_config.json`.
Example:
```json
"osmcp": {
  "command": "osmcp",
  "args": ["--policy", "/path/to/project/.osmcp/policy.toml"]
}
```

### 2. Demo GIF / Video
Create a placeholder in `README.md` for a demo GIF (e.g., `![osmcp Demo](assets/demo.gif)`) that shows Claude Desktop safely editing a file and being blocked by the policy engine when attempting to read outside `allowed_root`.
*(If the agent has access to a browser/terminal recorder, generate this asset. Otherwise, leave the placeholder and instructions for the user).*

### 3. Anthropic MCP Directory PR Prep
Create a file at `docs/launch/anthropic_registry_submission.md` containing the exact JSON payload and description text needed to submit `osmcp` to the official [Anthropic MCP Servers Directory](https://github.com/anthropics/mcp-servers).

### 4. Smithery Integration (Optional)
Add instructions in the `README.md` for how users can install `osmcp` via `npx @smithery/cli install osmcp` (or equivalent) if applicable.

## Location
- `README.md`
- `docs/launch/anthropic_registry_submission.md`
