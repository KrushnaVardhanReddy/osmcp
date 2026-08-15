package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase4 "github.com/osmcp/osmcp/docs/contracts/phase-4"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvToolE2E(t *testing.T) {
	// Need to build the binary since we are in e2e
	// Tests are usually run with `make e2e` which builds it

	// Set test environment variables
	os.Setenv("TEST_E2E_KEY", "E2E_VAL1")
	os.Setenv("TEST_E2E_KEY2", "E2E_VAL2")
	os.Setenv("SECRET_E2E_KEY", "SECRET_VAL")

	t.Cleanup(func() {
		os.Unsetenv("TEST_E2E_KEY")
		os.Unsetenv("TEST_E2E_KEY2")
		os.Unsetenv("SECRET_E2E_KEY")
	})

	t.Run("Allowed variables", func(t *testing.T) {
		wrapper := setupMCPClient(t, getPolicyPath("get_env_allowed.toml"))
		defer wrapper.Close()

		req := mcp.CallToolRequest{}
		req.Params.Name = "get_env"
		req.Params.Arguments = map[string]interface{}{
			"keys": []string{"TEST_E2E_KEY"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := wrapper.client.CallTool(ctx, req)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if len(res.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(res.Content))
		}

		textContent, ok := res.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", res.Content[0])
		}

		var env contracts.Envelope
		if err := json.Unmarshal([]byte(textContent.Text), &env); err != nil {
			t.Fatalf("failed to parse envelope: %v", err)
		}

		if !env.OK {
			t.Fatalf("expected OK true, got error: %v", env.Error)
		}

		dataBytes, _ := json.Marshal(env.Data)
		var data contracts_phase4.GetEnvData
		json.Unmarshal(dataBytes, &data)

		if val, ok := data.Variables["TEST_E2E_KEY"]; !ok || val != "E2E_VAL1" {
			t.Errorf("expected TEST_E2E_KEY=E2E_VAL1, got %v", val)
		}

		if _, ok := data.Variables["TEST_E2E_KEY2"]; ok {
			t.Errorf("did not request TEST_E2E_KEY2, but it was returned")
		}
	})

	t.Run("All explicit variables when keys empty", func(t *testing.T) {
		wrapper := setupMCPClient(t, getPolicyPath("get_env_allowed.toml"))
		defer wrapper.Close()

		req := mcp.CallToolRequest{}
		req.Params.Name = "get_env"
		req.Params.Arguments = map[string]interface{}{}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := wrapper.client.CallTool(ctx, req)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		textContent := res.Content[0].(mcp.TextContent)
		var env contracts.Envelope
		json.Unmarshal([]byte(textContent.Text), &env)

		if !env.OK {
			t.Fatalf("expected OK true, got error: %v", env.Error)
		}

		dataBytes, _ := json.Marshal(env.Data)
		var data contracts_phase4.GetEnvData
		json.Unmarshal(dataBytes, &data)

		if val, ok := data.Variables["TEST_E2E_KEY"]; !ok || val != "E2E_VAL1" {
			t.Errorf("expected TEST_E2E_KEY=E2E_VAL1")
		}
		if val, ok := data.Variables["TEST_E2E_KEY2"]; !ok || val != "E2E_VAL2" {
			t.Errorf("expected TEST_E2E_KEY2=E2E_VAL2")
		}
	})

	t.Run("Empty allowlist returns error", func(t *testing.T) {
		wrapper := setupMCPClient(t, getPolicyPath("get_env_empty_allowlist.toml"))
		defer wrapper.Close()

		req := mcp.CallToolRequest{}
		req.Params.Name = "get_env"
		req.Params.Arguments = map[string]interface{}{}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := wrapper.client.CallTool(ctx, req)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		textContent := res.Content[0].(mcp.TextContent)
		var env contracts.Envelope
		json.Unmarshal([]byte(textContent.Text), &env)

		if env.OK {
			t.Fatalf("expected OK false, got true")
		}

		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected error code %v, got %v", contracts.ErrPolicyDenied, env.Error.Code)
		}
	})
}
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
