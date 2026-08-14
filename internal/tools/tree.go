package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

)

type treeTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewTreeTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &treeTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *treeTool) Name() string {
	return "tree"
}

func (t *treeTool) Description() string {
	return "Renders a visual directory tree."
}

func (t *treeTool) IsMutating() bool {
	return false
}

func (t *treeTool) Execute(ctx context.Context, args contracts_phase1.TreeArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating())
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	info, err := os.Lstat(args.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, fmt.Sprintf("path not found: %s", args.Path), false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("stat error: %v", err), false, meta)
	}

	maxMatches := t.policy.Limits().MaxMatches

	if !info.IsDir() {
		// INV-UTIL-06: tree on file — returns ok=true, single-entry tree.
		data := contracts_phase1.TreeData{
			Tree:  filepath.Base(args.Path),
			Dirs:  0,
			Files: 1,
		}
		if args.DirsOnly {
			data.Files = 0
			data.Tree = ""
		}
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Success(t.Name(), data, meta)
	}

	var sb strings.Builder
	sb.WriteString(filepath.Base(args.Path) + "\n")

	dirs := 0
	files := 0
	entriesCount := 0
	truncated := false
	maxDepth := args.MaxDepth
	if maxDepth < 1 {
		maxDepth = 3
	}

	var walkErr error

	// We need to build the tree manually so we can know if an entry is the last one in its directory.
	// filepath.WalkDir does not easily tell us if it's the last entry.
	// Let's implement a recursive function.

	var buildTree func(dir string, prefix string, depth int) bool
	buildTree = func(dir string, prefix string, depth int) bool {
		if depth > maxDepth {
			return true
		}

		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return true // just skip on read error
		}

		// Filter entries based on ShowHidden and DirsOnly
		var filtered []fs.DirEntry
		for _, e := range entries {
			name := e.Name()
			if !args.ShowHidden && strings.HasPrefix(name, ".") {
				continue
			}
			if args.DirsOnly && !e.IsDir() {
				continue
			}
			filtered = append(filtered, e)
		}

		for i, e := range filtered {
			if entriesCount >= maxMatches {
				truncated = true
				return false
			}

			isLast := (i == len(filtered)-1)
			name := e.Name()

			if e.IsDir() {
				dirs++
			} else {
				files++
			}
			entriesCount++

			var pointer string
			if isLast {
				pointer = "└── "
			} else {
				pointer = "├── "
			}

			sb.WriteString(prefix + pointer + name + "\n")

			if e.IsDir() {
				var nextPrefix string
				if isLast {
					nextPrefix = prefix + "    "
				} else {
					nextPrefix = prefix + "│   "
				}
				if !buildTree(filepath.Join(dir, name), nextPrefix, depth+1) {
					return false
				}
			}
		}

		return true
	}

	buildTree(args.Path, "", 1)

	treeStr := strings.TrimRight(sb.String(), "\n")
	if truncated {
		treeStr += "\n... (truncated)"
		meta.Truncated = true
	}

	data := contracts_phase1.TreeData{
		Tree:  treeStr,
		Dirs:  dirs, // Does not count root dir per common tree implementations
		Files: files,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	if walkErr != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, walkErr.Error(), false, meta)
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *treeTool) RegisterMCP(s *server.MCPServer) {

	mcpTool := mcp.NewTool("tree",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the root directory."),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum depth to display."),
		),
		mcp.WithBoolean("show_hidden",
			mcp.Description("Include files and dirs starting with a dot."),
		),
		mcp.WithBoolean("dirs_only",
			mcp.Description("If true, only show directories."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.TreeArgs{
			MaxDepth: 3,
			ShowHidden: false,
			DirsOnly: false,
		}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if maxDepth, ok := argsMap["max_depth"].(float64); ok {
				args.MaxDepth = int(maxDepth)
			}
			if showHidden, ok := argsMap["show_hidden"].(bool); ok {
				args.ShowHidden = showHidden
			}
			if dirsOnly, ok := argsMap["dirs_only"].(bool); ok {
				args.DirsOnly = dirsOnly
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
