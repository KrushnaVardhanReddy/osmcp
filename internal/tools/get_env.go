package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/docs/contracts/phase-4"
)

type getEnvTool struct {
	policyEngine    contracts.PolicyEngine
	envelopeBuilder contracts.EnvelopeBuilder
	envAllowlist    []string
}

// NewGetEnvTool creates a new tool instance.
func NewGetEnvTool(policyEngine contracts.PolicyEngine, envelopeBuilder contracts.EnvelopeBuilder, envAllowlist []string) contracts.Tool {
	return &getEnvTool{
		policyEngine:    policyEngine,
		envelopeBuilder: envelopeBuilder,
		envAllowlist:    envAllowlist,
	}
}

func (t *getEnvTool) Name() string {
	return "get_env"
}

func (t *getEnvTool) Description() string {
	return "Returns workspace environment variables, strictly filtered by the policy allowlist."
}

func (t *getEnvTool) IsMutating() bool {
	return false
}

func (t *getEnvTool) RegisterMCP(s *server.MCPServer) {
	tool := mcp.NewTool(t.Name(), mcp.WithDescription(t.Description()), mcp.WithArray("keys", mcp.Description("Optional list of environment variable keys. If empty, returns all explicitly allowed keys.")))

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var keys []string

		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if keysArg, ok := argsMap["keys"].([]interface{}); ok {
				for _, k := range keysArg {
					if strK, ok := k.(string); ok {
						keys = append(keys, strK)
					}
				}
			}
		}

		args := contracts_phase4.GetEnvArgs{
			Keys: keys,
		}

		env := t.execute(ctx, args)
		resBytes, err := json.Marshal(env)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}

		res := mcp.NewToolResultText(string(resBytes))
		res.IsError = !env.OK
		return res, nil
	})
}

func (t *getEnvTool) execute(ctx context.Context, args contracts_phase4.GetEnvArgs) contracts.Envelope {
	// 1. Check policy limits & basic evaluate
	if err := t.policyEngine.Evaluate(ctx, t.Name(), []string{}, t.IsMutating()); err != nil {
		return t.envelopeBuilder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, contracts.Meta{})
	}

	// 2. Build explicit allowlist map for fast lookup
	allowMap := make(map[string]bool)
	for _, key := range t.envAllowlist {
		allowMap[key] = true
	}

	// If no allowlist configured, fail securely as per spec
	if len(allowMap) == 0 {
		return t.envelopeBuilder.Failure(t.Name(), contracts.ErrPolicyDenied, "no environment variables are whitelisted", false, contracts.Meta{})
	}

	resultMap := make(map[string]string)

	if len(args.Keys) > 0 {
		for _, key := range args.Keys {
			if allowMap[key] {
				resultMap[key] = os.Getenv(key)
			}
		}
	} else {
		for key := range allowMap {
			resultMap[key] = os.Getenv(key)
		}
	}

	data := contracts_phase4.GetEnvData{
		Variables: resultMap,
	}

	return t.envelopeBuilder.Success(t.Name(), data, contracts.Meta{})
}
