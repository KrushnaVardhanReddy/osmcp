package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

type patchTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewPatchTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &patchTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *patchTool) Name() string {
	return "patch"
}

func (t *patchTool) Description() string {
	return "Applies a unified diff patch to a file."
}

func (t *patchTool) IsMutating() bool {
	return true
}

func (t *patchTool) Execute(ctx context.Context, args contracts_phase2.PatchRequest) contracts.Envelope {
	return t.Patch(args)
}

func (t *patchTool) Patch(req contracts_phase2.PatchRequest) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{ExecutionTimeMs: 0, Truncated: false}

	if req.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}
	if req.Diff == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "diff must not be empty", false, meta)
	}

	if err := t.policy.Evaluate(context.Background(), t.Name(), []string{req.Path}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	contentBytes, err := os.ReadFile(req.Path)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to read file: %v", err), false, meta)
	}
	content := string(contentBytes)

	// Parse diff
	files, _, err := gitdiff.Parse(strings.NewReader(req.Diff))
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, fmt.Sprintf("failed to parse diff: %v", err), false, meta)
	}

    if len(files) == 0 {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "no valid patches found in diff", false, meta)
    }

    if len(files) > 1 {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "multiple patches found, expected single file patch", false, meta)
    }

	// Apply diff
    var out strings.Builder
	err = gitdiff.Apply(&out, strings.NewReader(content), files[0])
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to apply patch: %v", err), false, meta)
	}

	info, _ := os.Stat(req.Path)
	err = os.WriteFile(req.Path, []byte(out.String()), info.Mode())
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to write patched file: %v", err), false, meta)
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), fmt.Sprintf("Successfully patched %s", req.Path), meta)
}

func (t *patchTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path to the file to be patched.")),
		mcp.WithString("diff", mcp.Required(), mcp.Description("The unified diff text to apply.")),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.PatchRequest{}

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if diff, ok := argsMap["diff"].(string); ok {
				args.Diff = diff
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
