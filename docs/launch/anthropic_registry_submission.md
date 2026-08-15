# Publishing to the Official MCP Registry

The official MCP registry (`registry.modelcontextprotocol.io`) is now managed via the `mcp-publisher` CLI tool rather than through Pull Requests on GitHub.

To publish `osmcp`, follow these steps on your local machine:

## 1. Install the Publisher CLI
If you are on macOS or Linux, install via Homebrew:
```bash
brew install mcp-publisher
```

## 2. Initialize the Server Metadata
Navigate to your `osmcp` project root and run:
```bash
mcp-publisher init
```
This will interactively prompt you for information and generate a `server.json` file.
* **Name Format:** Use reverse-DNS style: `io.github.krushnavardhanreddy/osmcp`
* **Description:** "A typed, policy-controlled OS capability layer for AI agents."

## 3. Authenticate with GitHub
Authenticate the publisher CLI with your GitHub account:
```bash
mcp-publisher login github
```
Follow the device flow instructions provided in your terminal.

## 4. Publish!
Once authenticated and your `server.json` is ready, publish it to the registry:
```bash
mcp-publisher publish
```

You can then verify your server is live by visiting [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io/).


