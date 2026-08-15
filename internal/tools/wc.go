package tools

import (
	"context"
	"fmt"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type wcTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewWcTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &wcTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *wcTool) Name() string {
	return "wc"
}

func (t *wcTool) Description() string {
	return "Count lines, words, and bytes in a file."
}

func (t *wcTool) IsMutating() bool {
	return false
}

func (t *wcTool) Execute(ctx context.Context, args contracts_phase1.WcArgs) contracts.Envelope {
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

	if info.IsDir() {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path is a directory", false, meta)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}
	defer file.Close()

	var lines, words, bytesCount int
	inWord := false

	buf := make([]byte, 32*1024) // 32KB buffer for efficient streaming
	for {
		n, err := file.Read(buf)
		if n > 0 {
			bytesCount += n
			for i := 0; i < n; i++ {
				c := buf[i]
				if c == '\n' {
					lines++
				}
				// Whitespace detection
				isSpace := c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
				if !isSpace {
					if !inWord {
						inWord = true
						words++
					}
				} else {
					inWord = false
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
		}
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	data := contracts_phase1.WcData{
		Lines: lines,
		Words: words,
		Bytes: bytesCount,
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *wcTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("wc",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the file."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.WcArgs{}

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
