package tools

import (
	"context"
	"fmt"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type LsArgsExtended struct {
	contracts_phase1.LsArgs
	Pattern string `json:"pattern,omitempty"`
}

type lsTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewLsTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &lsTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *lsTool) Name() string {
	return "ls"
}

func (t *lsTool) Description() string {
	return "List contents of a directory."
}

func (t *lsTool) IsMutating() bool {
	return false
}

func (t *lsTool) Execute(ctx context.Context, args LsArgsExtended) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	if args.Pattern != "" {
		_, err := filepath.Match(args.Pattern, "")
		if err != nil {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "invalid pattern: "+err.Error(), false, meta)
		}
	}

	resolvedPath, err := filepath.EvalSymlinks(args.Path)
	if err != nil {
		// If symlink evaluation fails, we still pass the original path to the policy engine to let it decide or deny.
		resolvedPath = args.Path
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{resolvedPath}, t.IsMutating()); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	// Double check if path exists after policy has approved it, if there was an earlier evaluation error or if EvalSymlinks missed something
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrNotFound, "path not found", false, meta)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, "path not found", false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	limits := t.policy.Limits()
	maxMatches := limits.MaxMatches
	if maxMatches <= 0 {
		maxMatches = 100 // fallback just in case
	}

	entries := make([]contracts_phase1.LsEntry, 0)
	count := 0

	if !info.IsDir() {
		// INV-FILE-08: ls on a file returns 1 entry for the file itself.
		match := true
		if args.Pattern != "" {
			match, _ = filepath.Match(args.Pattern, info.Name())
		}
		if match {
			entries = append(entries, t.makeEntry(info, resolvedPath))
			count = 1
		}
	} else {
		// It's a directory
		if !args.Recursive {
			// Non-recursive listing
			dirEntries, err := os.ReadDir(resolvedPath)
			if err != nil {
				meta.ExecutionTimeMs = time.Since(start).Milliseconds()
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
			}
			for _, e := range dirEntries {
				if count >= maxMatches {
					meta.Truncated = true
					break
				}
				if !args.ShowHidden && strings.HasPrefix(e.Name(), ".") {
					continue
				}

				match := true
				if args.Pattern != "" {
					match, _ = filepath.Match(args.Pattern, e.Name())
				}

				if !match {
					continue
				}

				eInfo, err := e.Info()
				if err != nil {
					continue
				}
				entries = append(entries, t.makeEntry(eInfo, filepath.Join(resolvedPath, e.Name())))
				count++
			}
		} else {
			// Recursive listing
			maxDepth := args.MaxDepth
			if maxDepth < 1 {
				maxDepth = 1
			}

			err = filepath.WalkDir(resolvedPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return filepath.SkipDir // skip on error
				}

				if path == resolvedPath {
					return nil // Skip the base directory itself in the output
				}

				// Calculate depth correctly accounting for paths without trailing slashes
				relPath, _ := filepath.Rel(resolvedPath, path)
				currentDepth := 0
				if relPath != "." && relPath != "" {
					currentDepth = strings.Count(relPath, string(os.PathSeparator)) + 1
				}

				if currentDepth > maxDepth {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				if !args.ShowHidden {
					// Check if any component in the path relative to resolvedPath is hidden
					relPath, _ := filepath.Rel(resolvedPath, path)
					components := strings.Split(relPath, string(os.PathSeparator))
					isHidden := false
					for _, comp := range components {
						if strings.HasPrefix(comp, ".") {
							isHidden = true
							break
						}
					}
					if isHidden {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}

				match := true
				if args.Pattern != "" {
					match, _ = filepath.Match(args.Pattern, d.Name())
				}

				if !match {
					return nil
				}

				if count >= maxMatches {
					meta.Truncated = true
					return filepath.SkipDir // Break out if we are at max
				}

				eInfo, err := d.Info()
				if err == nil {
					entries = append(entries, t.makeEntry(eInfo, path))
					count++
				}

				return nil
			})
			if err != nil {
				meta.ExecutionTimeMs = time.Since(start).Milliseconds()
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
			}
		}
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	data := contracts_phase1.LsData{
		Entries: entries,
		Count:   count,
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *lsTool) makeEntry(info os.FileInfo, path string) contracts_phase1.LsEntry {
	return contracts_phase1.LsEntry{
		Name:    info.Name(),
		Path:    path,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    info.Mode().String(),
	}
}

func (t *lsTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("ls",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the directory to list."),
		),
		mcp.WithBoolean("recursive",
			mcp.Description("If true, walks subdirectories."),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum depth for recursive listing."),
		),
		mcp.WithBoolean("show_hidden",
			mcp.Description("If true, includes files and directories starting with a dot (.)."),
		),
		mcp.WithString("pattern",
			mcp.Description("Optional glob pattern to filter entries by name (e.g. \"*.go\")."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := LsArgsExtended{
			LsArgs: contracts_phase1.LsArgs{
				Recursive:  false,
				MaxDepth:   1,
				ShowHidden: false,
			},
		}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if rec, ok := argsMap["recursive"].(bool); ok {
				args.Recursive = rec
			}
			if maxDepth, ok := argsMap["max_depth"].(float64); ok {
				args.MaxDepth = int(maxDepth)
			}
			if showHidden, ok := argsMap["show_hidden"].(bool); ok {
				args.ShowHidden = showHidden
			}
			if pattern, ok := argsMap["pattern"].(string); ok {
				args.Pattern = pattern
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
