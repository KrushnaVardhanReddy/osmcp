package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase3 "github.com/osmcp/osmcp/docs/contracts/phase-3"
	"github.com/stretchr/testify/require"
)

func TestRunScriptE2E(t *testing.T) {
	tempDir := t.TempDir()

	policyTOML := `
[policy]
allowed_root = "` + tempDir + `"
allowed_tools = ["run_script"]
allow_run_script = true
allow_mutation = true

[limits]
timeout_ms = 10000
max_output_bytes = 1024000

[run_script]
allow_network = true
`
	policyPath := filepath.Join(tempDir, "policy.toml")
	err := os.WriteFile(policyPath, []byte(policyTOML), 0644)
	require.NoError(t, err)

	client := setupMCPClient(t, policyPath)
	defer client.Close()

	t.Run("execute simple script", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = "run_script"
		req.Params.Arguments = map[string]interface{}{
			"interpreter": "bash",
			"script":      "echo 'hello from e2e' > " + filepath.Join(tempDir, "output.txt"),
			"working_dir": tempDir,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := client.client.CallTool(ctx, req)
		require.NoError(t, err)


		require.Len(t, res.Content, 1)
		textContent, ok := res.Content[0].(mcp.TextContent)
		require.True(t, ok)

		var env contracts.Envelope
		err = json.Unmarshal([]byte(textContent.Text), &env)
		require.NoError(t, err)
		require.True(t, env.OK)

		dataBytes, _ := json.Marshal(env.Data)
		var data contracts_phase3.RunScriptData
		err = json.Unmarshal(dataBytes, &data)
		require.NoError(t, err)


        require.Equal(t, 0, data.ExitCode)


		content, err := os.ReadFile(filepath.Join(tempDir, "output.txt"))
		require.NoError(t, err)
		require.Equal(t, "hello from e2e\n", string(content))
	})

	t.Run("execute denied by blocklist", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = "run_script"
		req.Params.Arguments = map[string]interface{}{
			"interpreter": "bash",
			"script":      "curl http://example.com",
			"working_dir": tempDir,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := client.client.CallTool(ctx, req)
		require.NoError(t, err)


		require.Len(t, res.Content, 1)
		textContent, ok := res.Content[0].(mcp.TextContent)
		require.True(t, ok)

		var env contracts.Envelope
		err = json.Unmarshal([]byte(textContent.Text), &env)
		require.NoError(t, err)
		require.False(t, env.OK)
		require.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
	})
}
