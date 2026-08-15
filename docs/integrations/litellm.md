# LiteLLM Integration Guide

LiteLLM is an open-source AI Gateway that acts as a single proxy endpoint for 140+ LLMs. With the addition of LiteLLM's **MCP Gateway**, you can run `osmcp` as a centralized OS capability layer managed by LiteLLM.

By combining `osmcp`'s granular safety policies with LiteLLM's team-based routing and budgets, you can deploy AI agents securely across your organization.

## 1. Configure LiteLLM for `osmcp`

Add `osmcp` to your LiteLLM configuration file. Ensure you have built `osmcp` and have a `policy.toml` configured.

```yaml
# litellm_config.yaml

model_list:
  - model_name: gpt-4o
    litellm_params:
      model: openai/gpt-4o
      api_key: "os.environ/OPENAI_API_KEY"

mcp_servers:
  osmcp:
    command: "osmcp"
    args: ["--policy", "/absolute/path/to/policy.toml"]
    transport: "stdio"

general_settings:
  master_key: "sk-litellm-master"
```

Start your LiteLLM proxy:
```bash
litellm --config litellm_config.yaml
```

## 2. Using `osmcp` with Team Keys

LiteLLM allows you to generate API keys for specific teams. Because `osmcp` supports profile-based configurations (`osmcp --init --profile dev-agent`), you can easily map LiteLLM API keys to different agent capabilities.

While LiteLLM manages *who* can call the models and sets their *budget*, `osmcp` manages *what* they can actually do on the filesystem.

**Example Use Case:**
1.  **CI/CD Agent (Read-Only)**
    *   `osmcp` Policy: `allow_mutation = false`, `allowed_root = "/workspace/repo"`
    *   LiteLLM Key: `sk-ci-agent-...`
2.  **Dev Agent (Full Access)**
    *   `osmcp` Policy: `allow_mutation = true`, `allowed_tools = ["git_add", "patch", "run_script"]`
    *   LiteLLM Key: `sk-dev-agent-...`

## 3. Retrieval-Augmented Context (RAC) & Token Optimization

Since `osmcp` frequently fetches large log files, grep outputs, and diffs, running it through LiteLLM allows you to utilize **Smart Routing**.

For instance, you can configure LiteLLM to route simple, read-only `osmcp` interactions (like `ls` or `grep`) to a fast, cheap local model (like Ollama/Llama-3), and only escalate to `gpt-4o` or `claude-3.5-sonnet` when `osmcp` fetches complex code chunks that require high-reasoning capability.

## 4. Audit & Observability

Both `osmcp` and LiteLLM have native audit capabilities that complement each other:

- **LiteLLM Audit:** Tracks LLM latency, token cost, and model routing choices (e.g., via Langfuse or Helicone).
- **osmcp Audit:** Tracks the exact OS commands, path traversals, and mutating parameters inside the sandboxed environment (via NDJSON logs).

Together, they provide a 100% end-to-end transparent view of an autonomous agent's lifecycle.
