package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/itchyny/gojq"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

)

type jqTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewJqTool creates a new jq Tool.
func NewJqTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &jqTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *jqTool) Name() string {
	return "jq"
}

func (t *jqTool) Description() string {
	return "Query and filter JSON using jq syntax."
}

func (t *jqTool) IsMutating() bool {
	return false
}

func (t *jqTool) Execute(ctx context.Context, args contracts_phase1.JqArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{}

	err := t.policy.Evaluate(ctx, t.Name(), []string{}, t.IsMutating())
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	limits := t.policy.Limits()

	if int64(len(args.Input)) > limits.MaxOutputBytes {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Input exceeds MaxOutputBytes", false, meta)
	}

	var input interface{}
	if jsonErr := json.Unmarshal([]byte(args.Input), &input); jsonErr != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Invalid JSON input", false, meta)
	}

	query, err := gojq.Parse(args.Filter)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, fmt.Sprintf("Invalid jq filter: %v", err), false, meta)
	}

	iter := query.Run(input)
	var results []interface{}
	outputType := "null"

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, fmt.Sprintf("Invalid jq filter execution: %v", err), false, meta)
		}

		if len(results) == 0 {
			switch v.(type) {
			case string:
				outputType = "string"
			case float64, int:
				outputType = "number"
			case bool:
				outputType = "boolean"
			case []interface{}:
				outputType = "array"
			case map[string]interface{}:
				outputType = "object"
			case nil:
				outputType = "null"
			default:
				outputType = "unknown"
			}
		}

		results = append(results, v)
	}

	var outBytes []byte
	var marshalErr error

	if len(results) == 1 {
		// If there is only one result, don't wrap it in a JSON array unless we have to,
        // to match standard jq behaviour, but actually wait, we just marshal the result directly.
        if !args.Compact {
            outBytes, marshalErr = json.MarshalIndent(results[0], "", "  ")
        } else {
            outBytes, marshalErr = json.Marshal(results[0])
        }
	} else {
        if !args.Compact {
            outBytes, marshalErr = json.MarshalIndent(results, "", "  ")
        } else {
            outBytes, marshalErr = json.Marshal(results)
        }
	}

	if marshalErr != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to marshal output: %v", marshalErr), false, meta)
	}

	outStr := string(outBytes)

	if int64(len(outStr)) > limits.MaxOutputBytes {
		outStr = outStr[:limits.MaxOutputBytes]
		meta.Truncated = true
	}

	data := contracts_phase1.JqData{
		Result:     outStr,
		OutputType: outputType,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}

func (t *jqTool) RegisterMCP(s *server.MCPServer) {

	mcpTool := mcp.NewTool("jq",
		mcp.WithDescription(t.Description()),
		mcp.WithString("input",
			mcp.Required(),
			mcp.Description("JSON string to query."),
		),
		mcp.WithString("filter",
			mcp.Required(),
			mcp.Description("A jq filter expression (e.g. '.users[] | .name')."),
		),
		mcp.WithBoolean("compact",
			mcp.Description("If true, output is compact (no pretty-print)."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.JqArgs{
			Compact: false,
		}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if input, ok := argsMap["input"].(string); ok {
				args.Input = input
			}
			if filter, ok := argsMap["filter"].(string); ok {
				args.Filter = filter
			}
			if compact, ok := argsMap["compact"].(bool); ok {
				args.Compact = compact
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
