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

func TestSortTool_E2E(t *testing.T) {
	c := setupMCPClient(t, getPolicyPath("readonly.toml"))
	defer c.Close()

	// Use fixtures path as the test root to ensure it falls under the policy allowed_root
	tempDir := t.TempDir()

	// Create a policy that allows reading from this tempDir
	policyToml := `
[policy]
allowed_root = "` + tempDir + `"
allowed_tools = ["sort"]
allow_mutation = false

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 1000

[audit]
destination = "stderr"
`
	policyPath := filepath.Join(tempDir, "policy.toml")
	os.WriteFile(policyPath, []byte(policyToml), 0644)

	cCustom := setupMCPClient(t, policyPath)
	defer cCustom.Close()


	filePath := filepath.Join(tempDir, "sort_test.txt")
	content := "banana\napple\ncherry\napple\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("Plain Sort", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req := mcp.CallToolRequest{}
		req.Params.Name = "sort"
		req.Params.Arguments = map[string]interface{}{
			"path": filePath,
		}

		res, err := cCustom.client.CallTool(ctx, req)
		assert.NoError(t, err)
		assert.False(t, res.IsError)
		assert.Len(t, res.Content, 1)

		txt := res.Content[0].(mcp.TextContent).Text

		var env contracts.Envelope
		if err := json.Unmarshal([]byte(txt), &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}

		if !env.OK {
			t.Fatalf("expected success, got error: %v", env.Error)
		}

		dataBytes, _ := json.Marshal(env.Data)
		var sortData contracts_phase2.SortData
		if err := json.Unmarshal(dataBytes, &sortData); err != nil {
			t.Fatalf("failed to unmarshal SortData: %v", err)
		}

		if sortData.Count != 4 {
			t.Errorf("expected count 4, got %d", sortData.Count)
		}
		if sortData.Lines[0] != "apple" || sortData.Lines[1] != "apple" || sortData.Lines[2] != "banana" || sortData.Lines[3] != "cherry" {
			t.Errorf("unexpected sort result: %v", sortData.Lines)
		}
	})

	t.Run("Unique Sort", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req := mcp.CallToolRequest{}
		req.Params.Name = "sort"
		req.Params.Arguments = map[string]interface{}{
			"path":   filePath,
			"unique": true,
		}

		res, err := cCustom.client.CallTool(ctx, req)
		assert.NoError(t, err)
		assert.False(t, res.IsError)
		assert.Len(t, res.Content, 1)

		txt := res.Content[0].(mcp.TextContent).Text

		var env contracts.Envelope
		if err := json.Unmarshal([]byte(txt), &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}

		dataBytes, _ := json.Marshal(env.Data)
		var sortData contracts_phase2.SortData
		if err := json.Unmarshal(dataBytes, &sortData); err != nil {
			t.Fatalf("failed to unmarshal SortData: %v", err)
		}

		if sortData.Count != 3 {
			t.Errorf("expected count 3, got %d", sortData.Count)
		}
		if sortData.Lines[0] != "apple" || sortData.Lines[1] != "banana" || sortData.Lines[2] != "cherry" {
			t.Errorf("unexpected unique sort result: %v", sortData.Lines)
		}
	})
}
