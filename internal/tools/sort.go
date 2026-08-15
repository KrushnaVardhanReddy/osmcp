package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

type sortTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewSortTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &sortTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *sortTool) Name() string {
	return "sort"
}

func (t *sortTool) Description() string {
	return "Reads the contents of a file and returns its lines in sorted order."
}

func (t *sortTool) IsMutating() bool {
	return false
}

func (t *sortTool) Execute(ctx context.Context, args contracts_phase2.SortArgs) contracts.Envelope {
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

	// Check first 512 bytes for content type to prevent reading binary files
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err.Error() != "EOF" {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	contentType := http.DetectContentType(header[:n])
	isBinary := n > 0 && contentType != "text/plain; charset=utf-8" && contentType != "application/xml" && contentType != "application/json" && contentType != "application/javascript" && contentType != "application/xhtml+xml" && contentType != "text/xml; charset=utf-8" && contentType != "text/html; charset=utf-8"

	hasNullByte := false
	for _, b := range header[:n] {
		if b == 0 {
			hasNullByte = true
			break
		}
	}

	if isBinary || hasNullByte {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Cannot read binary file as text", false, meta)
	}

	if _, err := file.Seek(0, 0); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	if args.Numeric {
		sort.SliceStable(lines, func(i, j int) bool {
			valI, errI := strconv.ParseFloat(lines[i], 64)
			valJ, errJ := strconv.ParseFloat(lines[j], 64)

			if errI == nil && errJ == nil {
				return valI < valJ
			} else if errI == nil && errJ != nil {
				return true // Numeric first
			} else if errI != nil && errJ == nil {
				return false // Non-numeric last
			} else {
				return lines[i] < lines[j] // Both non-numeric, lexicographic
			}
		})
	} else {
		sort.Strings(lines)
	}

	if args.Reverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	if args.Unique && len(lines) > 0 {
		var uniqueLines []string
		uniqueLines = append(uniqueLines, lines[0])
		for i := 1; i < len(lines); i++ {
			if lines[i] != lines[i-1] {
				uniqueLines = append(uniqueLines, lines[i])
			}
		}
		lines = uniqueLines
	}

	maxOutputBytes := t.policy.Limits().MaxOutputBytes
	if maxOutputBytes <= 0 {
		maxOutputBytes = 1024 * 1024 // 1MB fallback
	}

	var finalLines []string
	if len(lines) > 0 {
		finalLines = make([]string, 0, len(lines))
	} else {
		finalLines = []string{}
	}
	currentBytes := int64(0)
	for i, line := range lines {
		// Calculate byte length: string length + 1 for newline if not last line (or just account for it)
		// For simplicity, we can serialize exactly how JSON will, but it's safe to just measure line length + 1
		lineLen := int64(len(line))
		if i > 0 {
			lineLen += 1 // representing the newline boundary
		}
		if currentBytes+lineLen > maxOutputBytes {
			meta.Truncated = true
			break
		}
		finalLines = append(finalLines, line)
		currentBytes += lineLen
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	data := contracts_phase2.SortData{
		Lines: finalLines,
		Count: len(finalLines),
	}

	return t.builder.Success(t.Name(), data, meta)
}

// Sort implements the contracts_phase2.SortTool interface.
func (t *sortTool) Sort(req contracts_phase2.SortArgs) contracts.Envelope {
	return t.Execute(context.Background(), req)
}

func (t *sortTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("sort",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the file to sort."),
		),
		mcp.WithBoolean("reverse",
			mcp.Description("If true, sort in descending order."),
		),
		mcp.WithBoolean("unique",
			mcp.Description("If true, deduplicate adjacent identical lines (sort -u)."),
		),
		mcp.WithBoolean("numeric",
			mcp.Description("If true, sort by numeric value instead of lexicographic order (sort -n)."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase2.SortArgs{
			Reverse: false,
			Unique:  false,
			Numeric: false,
		}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if rev, ok := argsMap["reverse"].(bool); ok {
				args.Reverse = rev
			}
			if uniq, ok := argsMap["unique"].(bool); ok {
				args.Unique = uniq
			}
			if num, ok := argsMap["numeric"].(bool); ok {
				args.Numeric = num
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
