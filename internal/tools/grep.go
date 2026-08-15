package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/tanqiangyes/grep-go/reader"
)

type grepTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewGrepTool creates a new grep Tool.
func NewGrepTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &grepTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *grepTool) Name() string {
	return "grep"
}

func (t *grepTool) Description() string {
	return "Search for patterns in files or directories."
}

func (t *grepTool) IsMutating() bool {
	return false
}

func (t *grepTool) Execute(ctx context.Context, args contracts_phase1.GrepArgs) contracts.Envelope {
	start := time.Now()

	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Pattern == "" || args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "pattern and path must not be empty", false, meta)
	}

	err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating())
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	info, err := os.Stat(args.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, fmt.Sprintf("path not found: %s", args.Path), false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("stat error: %v", err), false, meta)
	}

	policyLimits := t.policy.Limits()
	maxMatches := args.MaxMatches
	if maxMatches <= 0 || maxMatches > policyLimits.MaxMatches {
		maxMatches = policyLimits.MaxMatches
	}

	isRecursive := args.Recursive
	if !info.IsDir() {
		isRecursive = false
	}

	finder, err := reader.NewFinder(args.Pattern, !args.Literal, !args.CaseSensitive)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to create finder: %v", err), false, meta)
	}

	// For filtering files by Include and Exclude, we'll need to walk the directory manually or filter after grep-go.
	// grep-go's multiReader resolves all files using tools.Files(isRecursive, path).
	// We can't pass include/exclude directly to it.
	// We will get all files ourselves, filter them, and pass them to MultiReader.

	var searchPaths []string
	if info.IsDir() {
		walkErr := filepath.Walk(args.Path, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil // Ignore permission errors etc
			}
			if f.IsDir() {
				if !isRecursive && path != args.Path {
					return filepath.SkipDir
				}
				return nil
			}

			// Include logic
			if args.Include != "" {
				matched, err := filepath.Match(args.Include, f.Name())
				if err == nil && !matched {
					return nil
				}
			}

			// Exclude logic
			if args.Exclude != "" {
				matched, err := filepath.Match(args.Exclude, f.Name())
				if err == nil && matched {
					return nil
				}
			}
			searchPaths = append(searchPaths, path)
			return nil
		})
		if walkErr != nil {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to walk directory: %v", walkErr), false, meta)
		}
	} else {
		searchPaths = append(searchPaths, args.Path)
	}

	if len(searchPaths) == 0 {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Success(t.Name(), contracts_phase1.GrepData{Matches: []contracts_phase1.GrepMatch{}, Count: 0}, meta)
	}

	// Prevent timeout from hanging grep execution
	doneChan := make(chan struct{})
	var results []contracts_phase1.GrepMatch
	var execErr error
	count := 0
	truncated := false

	go func() {
		defer close(doneChan)
		read, err := reader.NewMultiReader(searchPaths, []reader.Finder{finder}, false)
		if err != nil {
			execErr = err
			return
		}

		read.Run()
		if read.IsError() != nil {
			execErr = read.IsError()
			return
		}

		res := read.Result()

	outer:
		for _, m := range res {
			for i, line := range m.Lines {
				if count >= maxMatches {
					truncated = true
					break outer
				}

				// The grep-go package doesn't support context lines natively.
				// However, spec says we need ContextBefore and ContextAfter (INV-GREP-09).
				// We need to implement manual context fetching if ContextLines > 0.
				var contextBefore []string
				var contextAfter []string

				if args.ContextLines > 0 {
					cb, ca := getContextLines(m.Filename, int(line), args.ContextLines)
					contextBefore = cb
					contextAfter = ca
				}

				results = append(results, contracts_phase1.GrepMatch{
					File:          m.Filename,
					Line:          int(line),
					Text:          strings.TrimRight(m.MatchString[i], "\r\n"),
					ContextBefore: contextBefore,
					ContextAfter:  contextAfter,
				})
				count++
			}
		}
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

	if execErr != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("grep error: %v", execErr), false, meta)
	}

	if results == nil {
		results = []contracts_phase1.GrepMatch{}
	}

	data := contracts_phase1.GrepData{
		Matches: results,
		Count:   count,
	}
	meta.Truncated = truncated

	return t.builder.Success(t.Name(), data, meta)
}

func getContextLines(filename string, lineNum int, contextLines int) ([]string, []string) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return []string{}, []string{}
	}
	lines := strings.Split(string(bytes), "\n")

	start := lineNum - 1 - contextLines
	if start < 0 {
		start = 0
	}

	end := lineNum - 1 + contextLines
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var cb []string
	for i := start; i < lineNum-1; i++ {
		cb = append(cb, strings.TrimRight(lines[i], "\r\n"))
	}

	var ca []string
	for i := lineNum; i <= end; i++ {
		ca = append(ca, strings.TrimRight(lines[i], "\r\n"))
	}

	if cb == nil {
		cb = []string{}
	}
	if ca == nil {
		ca = []string{}
	}

	return cb, ca
}

func (t *grepTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("grep",
		mcp.WithDescription(t.Description()),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Search pattern. Treated as a regex unless literal=true."),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to file or directory."),
		),
		mcp.WithBoolean("recursive",
			mcp.Description("Search subdirectories recursively."),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.Description("Case-sensitive search."),
		),
		mcp.WithBoolean("literal",
			mcp.Description("Treat pattern as a fixed string (not regex)."),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Lines of context around each match."),
		),
		mcp.WithString("include",
			mcp.Description("Glob to restrict searched files. e.g. '*.go'"),
		),
		mcp.WithString("exclude",
			mcp.Description("Glob to exclude files from search."),
		),
		mcp.WithNumber("max_matches",
			mcp.Description("Override policy max_matches for this call."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.GrepArgs{
			Recursive:     true,
			CaseSensitive: true,
			Literal:       false,
			ContextLines:  0,
		}

		// Parse arguments manually
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if pat, ok := argsMap["pattern"].(string); ok {
				args.Pattern = pat
			}
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if rec, ok := argsMap["recursive"].(bool); ok {
				args.Recursive = rec
			}
			if caseSens, ok := argsMap["case_sensitive"].(bool); ok {
				args.CaseSensitive = caseSens
			}
			if literal, ok := argsMap["literal"].(bool); ok {
				args.Literal = literal
			}
			if cl, ok := argsMap["context_lines"].(float64); ok {
				args.ContextLines = int(cl)
			}
			if inc, ok := argsMap["include"].(string); ok {
				args.Include = inc
			}
			if exc, ok := argsMap["exclude"].(string); ok {
				args.Exclude = exc
			}
			if max, ok := argsMap["max_matches"].(float64); ok {
				args.MaxMatches = int(max)
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
