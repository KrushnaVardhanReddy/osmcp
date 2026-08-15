package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpClientWrapper struct {
	client client.MCPClient
}

func setupMCPClient(t *testing.T, policyPath string) *mcpClientWrapper {
	cwd, _ := os.Getwd()
	binPath := filepath.Join(cwd, "..", "bin", "osmcp")

	mcpClient, err := client.NewStdioMCPClient(binPath, os.Environ(), "--policy", policyPath)
	if err != nil {
		t.Fatalf("failed to create stdio client: %v", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
	    // If it fails, let's try running binary manually to see stdout/stderr
	    out, _ := os.ReadFile("../error_log.txt") // Let's not rely on this, just print err
		t.Fatalf("failed to initialize MCP client: %v. Execution output: %s", err, string(out))
	}

	return &mcpClientWrapper{
		client: mcpClient,
	}
}

func (c *mcpClientWrapper) Close() {
	if c.client != nil {
		c.client.Close()
	}
}


func getPolicyPath(name string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "testdata", "policies", name)
}

func getFixturesPath() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "testdata", "fixtures")
}
