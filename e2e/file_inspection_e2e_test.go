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

func writeTempPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "temp_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["ls", "cat", "stat", "wc"]
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

func TestE2E_Ls_Success(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "ls"
	req.Params.Arguments = map[string]interface{}{
		"path": getFixturesPath(),
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

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.LsData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, data.Count, 1)
}

func TestE2E_Ls_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "ls"
	req.Params.Arguments = map[string]interface{}{
		"path": "/etc",
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

func TestE2E_Cat_Success(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "cat"
	req.Params.Arguments = map[string]interface{}{
		"path": filepath.Join(getFixturesPath(), "test.txt"),
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

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.CatData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, "hello", data.Content)
}

func TestE2E_Cat_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "cat"
	req.Params.Arguments = map[string]interface{}{
		"path": "/etc/passwd",
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

func TestE2E_Stat_Success(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "stat"
	req.Params.Arguments = map[string]interface{}{
		"path": filepath.Join(getFixturesPath(), "test.txt"),
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

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.StatData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", data.Name)
	assert.False(t, data.IsDir)
}

func TestE2E_Stat_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "stat"
	req.Params.Arguments = map[string]interface{}{
		"path": "/etc/passwd",
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

func TestE2E_Wc_Success(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "wc"
	req.Params.Arguments = map[string]interface{}{
		"path": filepath.Join(getFixturesPath(), "test.txt"),
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

	dataBytes, _ := json.Marshal(env.Data)
	var data contracts_phase1.WcData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, 1, data.Lines)
	assert.Equal(t, 1, data.Words)
	assert.Equal(t, 6, data.Bytes)
}

func TestE2E_Wc_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "wc"
	req.Params.Arguments = map[string]interface{}{
		"path": "/etc/passwd",
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
