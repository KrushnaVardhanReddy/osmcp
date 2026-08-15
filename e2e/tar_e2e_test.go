package e2e

import (
	"archive/tar"
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

func writeTarPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "readonly_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["tar"]
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

func createTestTar(t *testing.T, tempDir string) string {
	tarPath := filepath.Join(tempDir, "test.tar")
	file, err := os.Create(tarPath)
	assert.NoError(t, err)
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	content := []byte("hello world")
	hdr := &tar.Header{
		Name: "test.txt",
		Mode: 0600,
		Size: int64(len(content)),
	}
	err = tw.WriteHeader(hdr)
	assert.NoError(t, err)
	_, err = tw.Write(content)
	assert.NoError(t, err)

	return tarPath
}

func TestE2E_TarList_Success(t *testing.T) {
	tempRoot := t.TempDir()
	tarPath := createTestTar(t, tempRoot)
	c := setupMCPClient(t, writeTarPolicy(t, tempRoot))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tar"
	req.Params.Arguments = map[string]interface{}{
		"path":   tarPath,
		"action": "list",
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
	var data contracts_phase2.TarListData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, 1, data.Count)
	assert.Equal(t, "test.txt", data.Entries[0].Name)
}

func TestE2E_TarExtract_Success(t *testing.T) {
	tempRoot := t.TempDir()
	tarPath := createTestTar(t, tempRoot)
	c := setupMCPClient(t, writeTarPolicy(t, tempRoot))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "tar"
	req.Params.Arguments = map[string]interface{}{
		"path":   tarPath,
		"action": "extract",
		"entry":  "test.txt",
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
	var data contracts_phase2.TarExtractData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", data.Entry)
	assert.Equal(t, "hello world", data.Content)
}
