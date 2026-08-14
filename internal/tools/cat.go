package tools

import (
	"bufio"
	"context"
	"fmt"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type catTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewCatTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &catTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *catTool) Name() string {
	return "cat"
}

func (t *catTool) Description() string {
	return "Read the contents of a file."
}

func (t *catTool) IsMutating() bool {
	return false
}

func (t *catTool) Execute(ctx context.Context, args contracts_phase1.CatArgs) contracts.Envelope {
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

	// INV-FILE-04: cat Binary — Calling cat on a binary file returns ok=false, EXEC_FAILED.
	// Check first 512 bytes for content type
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err.Error() != "EOF" {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	contentType := http.DetectContentType(header[:n])
	// Text types or empty files
	isBinary := n > 0 && contentType != "text/plain; charset=utf-8" && contentType != "application/xml" && contentType != "application/json" && contentType != "application/javascript" && contentType != "application/xhtml+xml" && contentType != "text/xml; charset=utf-8" && contentType != "text/html; charset=utf-8"

	// A more robust check for non-text: check if any bytes are null (0x00), a common binary indicator
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

	// Seek back to start
	if _, err := file.Seek(0, 0); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	maxOutputBytes := t.policy.Limits().MaxOutputBytes
	if maxOutputBytes <= 0 {
		maxOutputBytes = 1024 * 1024 // 1MB fallback
	}

	var content []byte
	linesReturned := 0
	eofReached := false
	currentLine := 1

	startLine := args.StartLine
	if startLine < 1 {
		startLine = 1
	}

	scanner := bufio.NewScanner(file)
	// Optionally increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		if args.EndLine != nil && currentLine > *args.EndLine {
			break
		}

		if currentLine >= startLine {
			lineBytes := scanner.Bytes()
			lineLen := len(lineBytes)

			// Add 1 for newline if it's not the first line we're appending
			newlineLen := 0
			if len(content) > 0 {
				newlineLen = 1
			}

			if int64(len(content)+lineLen+newlineLen) > maxOutputBytes {
				meta.Truncated = true
				break
			}

			if newlineLen > 0 {
				content = append(content, '\n')
			}
			content = append(content, lineBytes...)
			linesReturned++
		}

		currentLine++
	}

	if err := scanner.Err(); err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, err.Error(), false, meta)
	}

	// Determine if EOF was genuinely reached based on whether scanner finished without breaking from limits
	if !meta.Truncated && (args.EndLine == nil || currentLine <= *args.EndLine) {
		eofReached = true
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	data := contracts_phase1.CatData{
		Content:       string(content),
		LinesReturned: linesReturned,
		EOFReached:    eofReached,
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *catTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("cat",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the file to read."),
		),
		mcp.WithNumber("start_line",
			mcp.Description("Line number to start reading from (1-indexed)."),
		),
		mcp.WithNumber("end_line",
			mcp.Description("Line number to stop reading at (inclusive). If omitted, reads to EOF."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase1.CatArgs{
			StartLine: 1,
		}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if sl, ok := argsMap["start_line"].(float64); ok {
				args.StartLine = int(sl)
			}
			if el, ok := argsMap["end_line"].(float64); ok {
				val := int(el)
				args.EndLine = &val
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
