package tools

import (
	"context"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

)

type diffTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewDiffTool creates a new diff Tool.
func NewDiffTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &diffTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *diffTool) Name() string {
	return "diff"
}

func (t *diffTool) Description() string {
	return "Unified diff between two strings."
}

func (t *diffTool) IsMutating() bool {
	return false
}

func (t *diffTool) Execute(ctx context.Context, args contracts_phase1.DiffArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{}

	err := t.policy.Evaluate(ctx, t.Name(), []string{}, t.IsMutating())
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	limits := t.policy.Limits()

	if int64(len(args.A)) > limits.MaxOutputBytes || int64(len(args.B)) > limits.MaxOutputBytes {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Input exceeds MaxOutputBytes", false, meta)
	}

	dmp := diffmatchpatch.New()

	// Create the diff. Using line mode is better for a unified patch appearance,
	// but diffmatchpatch's DiffMain works best on strings.
	diffs := dmp.DiffMain(args.A, args.B, false)
	dmp.DiffCleanupSemantic(diffs) // clean up diffs

	additions := 0
	deletions := 0
	for _, d := range diffs {
		if d.Type == diffmatchpatch.DiffInsert {
			additions++
		} else if d.Type == diffmatchpatch.DiffDelete {
			deletions++
		}
	}

	identical := (additions == 0 && deletions == 0)

	var patchText string
	if !identical {
        if args.ContextLines > 0 {
		    dmp.PatchMargin = args.ContextLines
        } else {
            dmp.PatchMargin = 3 // default
        }
		patches := dmp.PatchMake(args.A, diffs)
		patchText = dmp.PatchToText(patches)
	}

	if int64(len(patchText)) > limits.MaxOutputBytes {
		patchText = patchText[:limits.MaxOutputBytes]
		meta.Truncated = true
	}

	data := contracts_phase1.DiffData{
		Patch:     patchText,
		Additions: additions,
		Deletions: deletions,
		Identical: identical,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}

func (t *diffTool) RegisterMCP(s *server.MCPServer) {

	mcpTool := mcp.NewTool("diff",
		mcp.WithDescription(t.Description()),
		mcp.WithString("a",
			mcp.Required(),
			mcp.Description("The original text (left side)."),
		),
		mcp.WithString("b",
			mcp.Required(),
			mcp.Description("The modified text (right side)."),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Number of context lines around each change."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.DiffArgs{
			ContextLines: 3,
		}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if a, ok := argsMap["a"].(string); ok {
				args.A = a
			}
			if b, ok := argsMap["b"].(string); ok {
				args.B = b
			}
			if cl, ok := argsMap["context_lines"].(float64); ok {
				args.ContextLines = int(cl)
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
