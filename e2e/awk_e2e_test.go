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
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

func writeAwkPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "readonly_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["awk"]
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

func TestE2E_Awk_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeAwkPolicy(t, tempRoot))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("1 2\n3 4\n"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "awk"
	req.Params.Arguments = map[string]interface{}{
		"path":    filePath,
		"program": "{print $2}",
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
	var data contracts_phase2.AwkData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, "2\n4\n", data.Output)
}
