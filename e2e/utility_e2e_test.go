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

func writeUtilityTempPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "utility_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["tree", "head", "tail", "du"]
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

func TestE2E_Tree_Success(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tree"
	req.Params.Arguments = map[string]interface{}{
		"path": getFixturesPath(),
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
	var data contracts_phase1.TreeData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Greater(t, len(data.Tree), 0)
	assert.GreaterOrEqual(t, data.Files, 2)
}

func TestE2E_Tree_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tree"
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

func TestE2E_Head_Success(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "head"
	req.Params.Arguments = map[string]interface{}{
		"path": filepath.Join(getFixturesPath(), "sample.txt"),
		"lines": 2,
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
	var data contracts_phase1.HeadData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, 2, data.LinesReturned)
	assert.Contains(t, data.Content, "this is a sample text file.")
}

func TestE2E_Head_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "head"
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


func TestE2E_Tail_Success(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tail"
	req.Params.Arguments = map[string]interface{}{
		"path": filepath.Join(getFixturesPath(), "sample.txt"),
		"lines": 2,
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
	var data contracts_phase1.TailData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, 2, data.LinesReturned)
	assert.Contains(t, data.Content, "line 7 is here.")
}

func TestE2E_Tail_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tail"
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


func TestE2E_Du_Success(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "du"
	req.Params.Arguments = map[string]interface{}{
		"path": getFixturesPath(),
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
	var data contracts_phase1.DuData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Greater(t, data.TotalBytes, int64(0))
	assert.NotEmpty(t, data.Breakdown)
}

func TestE2E_Du_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeUtilityTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "du"
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
