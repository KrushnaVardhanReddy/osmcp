package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase4 "github.com/osmcp/osmcp/docs/contracts/phase-4"
)

func writeHashFileTempPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "temp_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["hash_file"]
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

func TestE2E_HashFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "hello.txt")
	content := []byte("e2e hash file test content")
	err := os.WriteFile(testFile, content, 0644)
	assert.NoError(t, err)

	sha256Hash := sha256.Sum256(content)
	sha256Expected := hex.EncodeToString(sha256Hash[:])

	c := setupMCPClient(t, writeHashFileTempPolicy(t, tmpDir))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "hash_file"
	req.Params.Arguments = map[string]interface{}{
		"path": testFile,
		"algorithm": "sha256",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.NotNil(t, res.Content)
	assert.Greater(t, len(res.Content), 0)

	textResult, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textResult.Text), &env)
	assert.NoError(t, err)
	assert.True(t, env.OK)
	assert.Nil(t, env.Error)

	dataMap, ok := env.Data.(map[string]interface{})
	assert.True(t, ok)

	dataBytes, _ := json.Marshal(dataMap)
	var data contracts_phase4.HashFileData
	json.Unmarshal(dataBytes, &data)

	assert.Equal(t, sha256Expected, data.Hash)
	assert.Equal(t, "sha256", data.Algorithm)
}

func TestE2E_HashFile_PolicyDenied(t *testing.T) {
	c := setupMCPClient(t, writeHashFileTempPolicy(t, t.TempDir()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "hash_file"
	req.Params.Arguments = map[string]interface{}{
		"path": "/etc/passwd",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.True(t, res.IsError)

	textResult, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	var env contracts.Envelope
	err = json.Unmarshal([]byte(textResult.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.NotNil(t, env.Error)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}
