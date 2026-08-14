package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

func setupGitRepo(t *testing.T) string {
	repoPath := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	err := cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoPath, "file.txt"), []byte("initial\ncontent"), 0644)
	assert.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoPath, "file.txt"), []byte("modified\ncontent"), 0644)
	assert.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "second commit")
	cmd.Dir = repoPath
	err = cmd.Run()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoPath, "file.txt"), []byte("uncommitted\ncontent"), 0644)
	assert.NoError(t, err)

	return repoPath
}

func writeGitTempPolicy(t *testing.T, allowedRoot string) string {
	policyPath := filepath.Join(t.TempDir(), "git_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["git_status", "git_diff", "git_log"]
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

func TestE2E_GitStatus_Success(t *testing.T) {
	repoPath := setupGitRepo(t)
	c := setupMCPClient(t, writeGitTempPolicy(t, repoPath))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_status"
	req.Params.Arguments = map[string]interface{}{
		"path": repoPath,
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
	var data contracts_phase1.GitStatusData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.False(t, data.Clean)
	assert.Len(t, data.Unstaged, 1)
	assert.Equal(t, "file.txt", data.Unstaged[0].Path)
}

func TestE2E_GitStatus_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeGitTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_status"
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

func TestE2E_GitDiff_Success(t *testing.T) {
	repoPath := setupGitRepo(t)
	c := setupMCPClient(t, writeGitTempPolicy(t, repoPath))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_diff"
	req.Params.Arguments = map[string]interface{}{
		"path": repoPath,
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
	var data contracts_phase1.GitDiffData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Len(t, data.Patches, 1)
	assert.Equal(t, "file.txt", data.Patches[0].File)
}

func TestE2E_GitDiff_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeGitTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_diff"
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

func TestE2E_GitLog_Success(t *testing.T) {
	repoPath := setupGitRepo(t)
	c := setupMCPClient(t, writeGitTempPolicy(t, repoPath))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_log"
	req.Params.Arguments = map[string]interface{}{
		"path": repoPath,
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
	var data contracts_phase1.GitLogData
	err = json.Unmarshal(dataBytes, &data)
	assert.NoError(t, err)
	assert.Equal(t, 2, data.Count)
	assert.Equal(t, "second commit\n", data.Commits[0].Message)
}

func TestE2E_GitLog_PolicyDenial(t *testing.T) {
	c := setupMCPClient(t, writeGitTempPolicy(t, getFixturesPath()))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_log"
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
