package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

// -----------------------------------------------------------------------------
// write_file Tool
// -----------------------------------------------------------------------------

type writeFileTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewWriteFileTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &writeFileTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *writeFileTool) Name() string {
	return "write_file"
}

func (t *writeFileTool) Description() string {
	return "Creates a new file or overwrites an existing file with the provided content."
}

func (t *writeFileTool) IsMutating() bool {
	return true
}

func (t *writeFileTool) Execute(ctx context.Context, args contracts_phase2.WriteFileRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}


	// Check if file exists when Overwrite is false
	if !args.Overwrite {
		if _, err := os.Stat(args.Path); err == nil {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "file already exists and overwrite is false", false, meta)
		} else if !os.IsNotExist(err) {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
		}
	}

	// Write file
	err := os.WriteFile(args.Path, []byte(args.Content), 0644)
	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully wrote %d bytes to %s", len(args.Content), args.Path), meta)
}

func (t *writeFileTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path to the file.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write.")),
		mcp.WithBoolean("overwrite", mcp.Description("If true, overwrite existing file (default: false).")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.WriteFileRequest{
			Overwrite: false,
		}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if content, ok := argsMap["content"].(string); ok {
				args.Content = content
			}
			if ow, ok := argsMap["overwrite"].(bool); ok {
				args.Overwrite = ow
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

// -----------------------------------------------------------------------------
// append_file Tool
// -----------------------------------------------------------------------------

type appendFileTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewAppendFileTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &appendFileTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *appendFileTool) Name() string {
	return "append_file"
}

func (t *appendFileTool) Description() string {
	return "Appends content to an existing file. Creates it if it doesn't exist."
}

func (t *appendFileTool) IsMutating() bool {
	return true
}

func (t *appendFileTool) Execute(ctx context.Context, args contracts_phase2.AppendFileRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}


	file, err := os.OpenFile(args.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}
	defer file.Close()

	n, err := file.WriteString(args.Content)
	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully appended %d bytes to %s", n, args.Path), meta)
}

func (t *appendFileTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path to the file.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to append.")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.AppendFileRequest{}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if content, ok := argsMap["content"].(string); ok {
				args.Content = content
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
