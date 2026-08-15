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
)

func writePatchPolicy(t *testing.T, allowedRoot string, allowMutation bool) string {
	policyPath := filepath.Join(t.TempDir(), "mutation_policy.toml")
	allowMut := "false"
	if allowMutation {
		allowMut = "true"
	}
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["patch"]
allow_mutation = `+allowMut+`

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

func TestE2E_Patch_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writePatchPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello\n"), 0644)

	diff := `--- test.txt
+++ test.txt
@@ -1 +1 @@
-hello
+hello world
`

	req := mcp.CallToolRequest{}
	req.Params.Name = "patch"
	req.Params.Arguments = map[string]interface{}{
		"path": filePath,
		"diff": diff,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "hello world\n", string(content))
}

func TestE2E_Patch_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writePatchPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello\n"), 0644)

	diff := `--- test.txt
+++ test.txt
@@ -1 +1 @@
-hello
+hello world
`

	req := mcp.CallToolRequest{}
	req.Params.Name = "patch"
	req.Params.Arguments = map[string]interface{}{
		"path": filePath,
		"diff": diff,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(content))
}
