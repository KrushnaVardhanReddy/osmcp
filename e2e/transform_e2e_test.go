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

func writeTransformTempPolicy(t *testing.T) string {
	policyPath := filepath.Join(t.TempDir(), "transform_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "/tmp"
allowed_tools = ["jq", "sed", "diff"]
allow_mutation = false

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 100

[audit]
destination = "stderr"
`), 0644)
	assert.NoError(t, err)
	return policyPath
}

func TestE2E_Jq_Success(t *testing.T) {
	c := setupMCPClient(t, writeTransformTempPolicy(t))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "jq"
	req.Params.Arguments = map[string]interface{}{
		"input": `{"name":"test","value":123}`,
		"filter": ".name",
		"compact": true,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.True(t, env.OK)

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.JqData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, `"test"`, data.Result)
	assert.Equal(t, "string", data.OutputType)
}

func TestE2E_Jq_InvalidArgs(t *testing.T) {
	c := setupMCPClient(t, writeTransformTempPolicy(t))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "jq"
	req.Params.Arguments = map[string]interface{}{
		"input": `{"name":"test"`,
		"filter": ".name",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestE2E_Sed_Success(t *testing.T) {
	c := setupMCPClient(t, writeTransformTempPolicy(t))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "sed"
	req.Params.Arguments = map[string]interface{}{
		"input": "hello world",
		"expression": "s/world/universe/",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.True(t, env.OK)

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.SedData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, "hello universe", data.Result)
	assert.Equal(t, 1, data.ReplacementsMade)
}

func TestE2E_Sed_InvalidArgs(t *testing.T) {
	c := setupMCPClient(t, writeTransformTempPolicy(t))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "sed"
	req.Params.Arguments = map[string]interface{}{
		"input": "hello world",
		"expression": "s/world/universe", // missing trailing slash
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestE2E_Diff_Success(t *testing.T) {
	c := setupMCPClient(t, writeTransformTempPolicy(t))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "diff"
	req.Params.Arguments = map[string]interface{}{
		"a": "hello world\nhow are you\n",
		"b": "hello universe\nhow are you\n",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.True(t, env.OK)

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.DiffData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.False(t, data.Identical)
	assert.Contains(t, data.Patch, "-world")
	assert.Contains(t, data.Patch, "+universe")
}
