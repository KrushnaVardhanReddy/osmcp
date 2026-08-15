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

func writeFindTempPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "find_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["find"]
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

func TestE2E_Find_Success(t *testing.T) {
	c := setupMCPClient(t, writeFindTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "find"
	req.Params.Arguments = map[string]interface{}{
		"path": getFixturesPath(),
		"name": "*.txt",
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
	var data contracts_phase1.FindData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)

	// We have sample.txt and test.txt
	assert.GreaterOrEqual(t, data.Count, 2)
}

func TestE2E_Find_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeFindTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "find"
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
