package tools

import (
	"context"
	"fmt"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"os"
	"path/filepath"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type statTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewStatTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &statTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *statTool) Name() string {
	return "stat"
}

func (t *statTool) Description() string {
	return "Get metadata for a file or directory."
}

func (t *statTool) IsMutating() bool {
	return false
}

func (t *statTool) Execute(ctx context.Context, args contracts_phase1.StatArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	resolvedPath, err := filepath.EvalSymlinks(args.Path)
	if err != nil {
		resolvedPath = args.Path
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{resolvedPath}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, "path not found", false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	data := contracts_phase1.StatData{
		Name:    info.Name(),
		Path:    resolvedPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    info.Mode().String(),
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}

func (t *statTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("stat",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the file or directory."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.StatArgs{}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
		}

		envelope := t.Execute(ctx, args)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resBytes)), nil
	})
}
