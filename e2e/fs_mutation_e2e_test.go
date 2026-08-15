package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"encoding/json"
	"github.com/stretchr/testify/assert"
)

func writeMutationPolicy(t *testing.T, allowedRoot string, allowMutation bool) string {
	policyPath := filepath.Join(t.TempDir(), "mutation_policy.toml")
	allowMut := "false"
	if allowMutation {
		allowMut = "true"
	}
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["write_file", "append_file", "mkdir", "rm", "mv", "cp"]
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

func TestE2E_WriteFile_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	req := mcp.CallToolRequest{}
	req.Params.Name = "write_file"
	req.Params.Arguments = map[string]interface{}{
		"path":    filePath,
		"content": "hello world",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestE2E_WriteFile_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	req := mcp.CallToolRequest{}
	req.Params.Name = "write_file"
	req.Params.Arguments = map[string]interface{}{
		"path":    filePath,
		"content": "hello world",
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

	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestE2E_AppendFile_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello "), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "append_file"
	req.Params.Arguments = map[string]interface{}{
		"path":    filePath,
		"content": "world",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestE2E_AppendFile_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello "), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "append_file"
	req.Params.Arguments = map[string]interface{}{
		"path":    filePath,
		"content": "world",
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
	assert.Equal(t, "hello ", string(content))
}

func TestE2E_Mkdir_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dirPath := filepath.Join(tempRoot, "new_dir")
	req := mcp.CallToolRequest{}
	req.Params.Name = "mkdir"
	req.Params.Arguments = map[string]interface{}{
		"path": dirPath,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	info, err := os.Stat(dirPath)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestE2E_Mkdir_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dirPath := filepath.Join(tempRoot, "new_dir")
	req := mcp.CallToolRequest{}
	req.Params.Name = "mkdir"
	req.Params.Arguments = map[string]interface{}{
		"path": dirPath,
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

	_, err = os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err))
}

func TestE2E_Rm_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "rm"
	req.Params.Arguments = map[string]interface{}{
		"path": filePath,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestE2E_Rm_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	filePath := filepath.Join(tempRoot, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "rm"
	req.Params.Arguments = map[string]interface{}{
		"path": filePath,
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

	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestE2E_Mv_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	srcPath := filepath.Join(tempRoot, "src.txt")
	dstPath := filepath.Join(tempRoot, "dst.txt")
	os.WriteFile(srcPath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "mv"
	req.Params.Arguments = map[string]interface{}{
		"source":      srcPath,
		"destination": dstPath,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))

	content, err := os.ReadFile(dstPath)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestE2E_Mv_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	srcPath := filepath.Join(tempRoot, "src.txt")
	dstPath := filepath.Join(tempRoot, "dst.txt")
	os.WriteFile(srcPath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "mv"
	req.Params.Arguments = map[string]interface{}{
		"source":      srcPath,
		"destination": dstPath,
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

	_, err = os.Stat(srcPath)
	assert.NoError(t, err)

	_, err = os.Stat(dstPath)
	assert.True(t, os.IsNotExist(err))
}

func TestE2E_Cp_Success(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	srcPath := filepath.Join(tempRoot, "src.txt")
	dstPath := filepath.Join(tempRoot, "dst.txt")
	os.WriteFile(srcPath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "cp"
	req.Params.Arguments = map[string]interface{}{
		"source":      srcPath,
		"destination": dstPath,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	_, err = os.Stat(srcPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(dstPath)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestE2E_Cp_PolicyDenial(t *testing.T) {
	tempRoot := t.TempDir()
	c := setupMCPClient(t, writeMutationPolicy(t, tempRoot, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	srcPath := filepath.Join(tempRoot, "src.txt")
	dstPath := filepath.Join(tempRoot, "dst.txt")
	os.WriteFile(srcPath, []byte("hello"), 0644)

	req := mcp.CallToolRequest{}
	req.Params.Name = "cp"
	req.Params.Arguments = map[string]interface{}{
		"source":      srcPath,
		"destination": dstPath,
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

	_, err = os.Stat(dstPath)
	assert.True(t, os.IsNotExist(err))
}
