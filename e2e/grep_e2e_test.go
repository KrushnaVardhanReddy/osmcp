package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)



func TestE2E_ToolsList_Visible(t *testing.T) {
	c := setupMCPClient(t, getPolicyPath("readonly.toml"))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	assert.NoError(t, err)

	foundGrep := false
	for _, tool := range res.Tools {
		if tool.Name == "grep" {
			foundGrep = true
		}
	}
	assert.True(t, foundGrep, "grep tool should be visible")
}

func TestE2E_ToolsList_NotVisible(t *testing.T) {
	c := setupMCPClient(t, getPolicyPath("no_grep.toml"))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	assert.NoError(t, err)

	foundGrep := false
	for _, tool := range res.Tools {
		if tool.Name == "grep" {
			foundGrep = true
		}
	}
	assert.False(t, foundGrep, "grep tool should NOT be visible")
}

func TestE2E_Grep_Success(t *testing.T) {
    // Override the policy for success test, as readonly.toml's allowed_root is /home/user/myproject
    // We create a temp policy that uses the current workspace.
    policyPath := filepath.Join(t.TempDir(), "success_policy.toml")
    err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+getFixturesPath()+`"
allowed_tools = ["grep"]
allow_mutation = false

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 100

[audit]
destination = "stderr"
    `), 0644)
    assert.NoError(t, err)

	c := setupMCPClient(t, policyPath)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "grep"
	req.Params.Arguments = map[string]interface{}{
		"pattern": "TODO",
		"path":    getFixturesPath(),
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Len(t, res.Content, 1)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.True(t, env.OK)

	// Convert env.Data from map to GrepData
	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.GrepData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)

	assert.Equal(t, 2, data.Count)
}

func TestE2E_Grep_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, getPolicyPath("readonly.toml"))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "grep"
	req.Params.Arguments = map[string]interface{}{
		"pattern": "root",
		"path":    "/etc/passwd", // outside allowed_root
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}
