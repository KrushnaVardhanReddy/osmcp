package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

// ErrTruncated is an internal sentinel error to signal that the match limit was reached.
// We use filepath.SkipDir to skip directories, but we need a way to stop the walk entirely.
var ErrTruncated = errors.New("truncated max matches reached")

type findTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewFindTool creates a new find Tool.
func NewFindTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &findTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *findTool) Name() string {
	return "find"
}

func (t *findTool) Description() string {
	return "Discover files by name pattern, type, size, or modification time."
}

func (t *findTool) IsMutating() bool {
	return false
}

func (t *findTool) Execute(ctx context.Context, args contracts_phase1.FindArgs) contracts.Envelope {
	start := time.Now()

	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if args.Name != nil && *args.Name != "" {
		_, err := filepath.Match(*args.Name, "")
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "invalid name glob pattern", false, meta)
		}
	}

	resolvedPath, err := filepath.EvalSymlinks(args.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, fmt.Sprintf("path not found: %s", args.Path), false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to evaluate symlinks: %v", err), false, meta)
	}

	err = t.policy.Evaluate(ctx, t.Name(), []string{resolvedPath}, t.IsMutating())
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	policyLimits := t.policy.Limits()
	maxMatches := policyLimits.MaxMatches
	if maxMatches <= 0 {
		maxMatches = 100 // Fallback
	}

	var results []contracts_phase1.FindEntry
	if results == nil {
		results = []contracts_phase1.FindEntry{}
	}

	count := 0
	truncated := false
	doneChan := make(chan struct{})
	var walkErr error

	go func() {
		defer close(doneChan)
		walkErr = filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				return nil // Ignore permissions errors and continue
			}

			// Depth calculation
			relPath, err := filepath.Rel(args.Path, path)
			if err != nil {
				return nil
			}

			depth := 0
			if relPath != "." {
				// Normalize to ensure we count standard separators
				normalizedRelPath := filepath.ToSlash(relPath)
				depth = strings.Count(normalizedRelPath, "/") + 1
			}

			if depth > args.MaxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Filter: Type
			isDir := d.IsDir()
			if args.Type == "file" && isDir {
				return nil
			}
			if args.Type == "dir" && !isDir {
				return nil
			}

			// Filter: Name
			if args.Name != nil && *args.Name != "" {
				matched, err := filepath.Match(*args.Name, d.Name())
				if err != nil || !matched {
					return nil
				}
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			// Filter: MinSize
			if args.MinSize != nil {
				if info.Size() < *args.MinSize {
					return nil
				}
			}

			// Filter: MaxSize
			if args.MaxSize != nil {
				if info.Size() > *args.MaxSize {
					return nil
				}
			}

			// Filter: ModifiedAfter
			if args.ModifiedAfter != nil {
				if !info.ModTime().After(*args.ModifiedAfter) {
					return nil
				}
			}

			// All filters passed, add to matches
			if count >= maxMatches {
				truncated = true
				return ErrTruncated
			}

			results = append(results, contracts_phase1.FindEntry{
				Path:    path,
				Name:    d.Name(),
				IsDir:   isDir,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
			count++

			return nil
		})
	}()

	select {
	case <-doneChan:
	case <-ctx.Done():
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrTimeout, "execution cancelled by context", false, meta)
	case <-time.After(time.Duration(policyLimits.TimeoutMs) * time.Millisecond):
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrTimeout, "execution exceeded timeout", false, meta)
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	meta.Truncated = truncated

	if walkErr != nil && walkErr != ErrTruncated {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("walk error: %v", walkErr), false, meta)
	}

	data := contracts_phase1.FindData{
		Matches: results,
		Count:   count,
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *findTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("find",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the directory to search."),
		),
		mcp.WithString("name",
			mcp.Description("Optional glob pattern to match file names (e.g. '*.go', 'main.*')."),
		),
		mcp.WithString("type",
			mcp.Description("Filter by entry type. Enum: file, dir, any."),
		),
		mcp.WithNumber("min_size",
			mcp.Description("Optional minimum file size in bytes (inclusive)."),
		),
		mcp.WithNumber("max_size",
			mcp.Description("Optional maximum file size in bytes (inclusive)."),
		),
		mcp.WithString("modified_after",
			mcp.Description("Optional ISO 8601 timestamp. Only returns files modified after this time."),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum directory depth to walk."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.FindArgs{
			Type:     "any",
			MaxDepth: 10,
		}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if name, ok := argsMap["name"].(string); ok {
				args.Name = &name
			}
			if typ, ok := argsMap["type"].(string); ok {
				args.Type = typ
			}
			if min, ok := argsMap["min_size"].(float64); ok {
				val := int64(min)
				args.MinSize = &val
			}
			if max, ok := argsMap["max_size"].(float64); ok {
				val := int64(max)
				args.MaxSize = &val
			}
			if modAfterStr, ok := argsMap["modified_after"].(string); ok {
				if modTime, err := time.Parse(time.RFC3339, modAfterStr); err == nil {
					args.ModifiedAfter = &modTime
				}
			}
			if md, ok := argsMap["max_depth"].(float64); ok {
				args.MaxDepth = int(md)
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
