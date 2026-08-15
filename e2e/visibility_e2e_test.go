package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestE2E_ToolsList_AllVisible(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "all_visible_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "/tmp"
allowed_tools = [
  "grep", "find", "ls", "cat", "head", "tail",
  "tree", "du", "wc", "stat",
  "git_status", "git_diff", "git_log",
  "sed", "jq", "diff"
]
allow_mutation = false

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 100

[audit]
destination = "stderr"
`), 0644)
	assert.NoError(t, err)

	c := setupMCPClient(t, policyPath)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	assert.NoError(t, err)

	toolNames := make(map[string]bool)
	for _, tool := range res.Tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"grep", "find", "ls", "cat", "head", "tail",
		"tree", "du", "wc", "stat",
		"git_status", "git_diff", "git_log",
		"sed", "jq", "diff",
	}

	for _, name := range expectedTools {
		assert.True(t, toolNames[name], "tool %s should be visible", name)
	}
}

func TestE2E_ToolsList_PartiallyVisible(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "partial_visible_policy.toml")
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "/tmp"
allowed_tools = [
  "grep", "ls", "cat", "git_status"
]
allow_mutation = false

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 100

[audit]
destination = "stderr"
`), 0644)
	assert.NoError(t, err)

	c := setupMCPClient(t, policyPath)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	assert.NoError(t, err)

	toolNames := make(map[string]bool)
	for _, tool := range res.Tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"grep", "ls", "cat", "git_status"}
	notExpectedTools := []string{"find", "tree", "jq", "git_diff"}

	for _, name := range expectedTools {
		assert.True(t, toolNames[name], "tool %s should be visible", name)
	}
	for _, name := range notExpectedTools {
		assert.False(t, toolNames[name], "tool %s should NOT be visible", name)
	}
}
