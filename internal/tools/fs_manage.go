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
// mkdir Tool
// -----------------------------------------------------------------------------

type mkdirTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewMkdirTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &mkdirTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *mkdirTool) Name() string {
	return "mkdir"
}

func (t *mkdirTool) Description() string {
	return "Creates a directory, automatically creating parent directories if they do not exist."
}

func (t *mkdirTool) IsMutating() bool {
	return true
}

func (t *mkdirTool) Execute(ctx context.Context, args contracts_phase2.MkdirRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}


	err := os.MkdirAll(args.Path, 0755)
	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully created directory %s", args.Path), meta)
}

func (t *mkdirTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path of the directory.")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.MkdirRequest{}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
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

// -----------------------------------------------------------------------------
// rm Tool
// -----------------------------------------------------------------------------

type rmTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewRmTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &rmTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *rmTool) Name() string {
	return "rm"
}

func (t *rmTool) Description() string {
	return "Removes a file or directory."
}

func (t *rmTool) IsMutating() bool {
	return true
}

func (t *rmTool) Execute(ctx context.Context, args contracts_phase2.RmRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}


	info, err := os.Stat(args.Path)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, "path not found", false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	if info.IsDir() && !args.Recursive {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "target is a directory and recursive is false", false, meta)
	}

	if args.Recursive {
		err = os.RemoveAll(args.Path)
	} else {
		err = os.Remove(args.Path)
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully removed %s", args.Path), meta)
}

func (t *rmTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path to remove.")),
		mcp.WithBoolean("recursive", mcp.Description("Must be true to remove directories (default: false).")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.RmRequest{
			Recursive: false,
		}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if rec, ok := argsMap["recursive"].(bool); ok {
				args.Recursive = rec
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
