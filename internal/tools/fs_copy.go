package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

// -----------------------------------------------------------------------------
// mv Tool
// -----------------------------------------------------------------------------

type mvTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewMvTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &mvTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *mvTool) Name() string {
	return "mv"
}

func (t *mvTool) Description() string {
	return "Moves or renames a file or directory."
}

func (t *mvTool) IsMutating() bool {
	return true
}

func (t *mvTool) Execute(ctx context.Context, args contracts_phase2.MvRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Source == "" || args.Destination == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "source and destination must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Source}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Destination}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}



	err := os.Rename(args.Source, args.Destination)
	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully moved %s to %s", args.Source, args.Destination), meta)
}

func (t *mvTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("source", mcp.Required(), mcp.Description("The path of the source item.")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("The path of the destination item.")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.MvRequest{}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if src, ok := argsMap["source"].(string); ok {
				args.Source = src
			}
			if dst, ok := argsMap["destination"].(string); ok {
				args.Destination = dst
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
// cp Tool
// -----------------------------------------------------------------------------

type cpTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewCpTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &cpTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *cpTool) Name() string {
	return "cp"
}

func (t *cpTool) Description() string {
	return "Copies a file or directory recursively."
}

func (t *cpTool) IsMutating() bool {
	return true
}

func (t *cpTool) Execute(ctx context.Context, args contracts_phase2.CpRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if args.Source == "" || args.Destination == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "source and destination must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Source}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{args.Destination}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}



	err := copyRecursive(args.Source, args.Destination)
	meta.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully copied %s to %s", args.Source, args.Destination), meta)
}

func (t *cpTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("source", mcp.Required(), mcp.Description("The path of the source item.")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("The path of the destination item.")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.CpRequest{}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if src, ok := argsMap["source"].(string); ok {
				args.Source = src
			}
			if dst, ok := argsMap["destination"].(string); ok {
				args.Destination = dst
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

func copyRecursive(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	// Prevent infinite recursion if dst is a subdirectory of src
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}

	// Ensure dst is not a subpath of src
	if len(absDst) >= len(absSrc) && absDst[:len(absSrc)] == absSrc {
		// If they are exactly the same, or dst has a separator immediately following the src prefix
		if len(absDst) == len(absSrc) || absDst[len(absSrc)] == filepath.Separator {
			return fmt.Errorf("cannot copy directory into itself")
		}
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
