package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

type awkTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewAwkTool creates a new awk Tool.
func NewAwkTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &awkTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *awkTool) Name() string {
	return "awk"
}

func (t *awkTool) Description() string {
	return "Runs an AWK program against a file."
}

func (t *awkTool) IsMutating() bool {
	return false
}

func (t *awkTool) Awk(args phase2.AwkArgs) contracts.Envelope {
	// The contract uses Awk(args phase2.AwkArgs) but we need context for policy evaluation.
	// Since AwkTool doesn't have ctx in its signature in phase2.AwkTool, we'll pass context.Background()
	// Or we can add Execute with context. Let's look at phase2.AwkTool contract.
	// Wait, the contract says Awk(req AwkArgs) contracts.Envelope
	// But PolicyEngine.Evaluate needs context.Context.
	// Let's use context.Background() inside Awk() since it's missing from the interface.
	return t.execute(context.Background(), args)
}

func (t *awkTool) execute(ctx context.Context, args phase2.AwkArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{}

	if args.Program == "" {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Program is required", false, meta)
	}

	if args.Path == "" {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Path is required", false, meta)
	}

	err := t.policy.Evaluate(ctx, t.Name(), []string{args.Path}, t.IsMutating())
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	// Fast path for empty path (checked above, but to be sure we are aligned with policy evaluation if that passed)

	// Security: block file writes
	if strings.Contains(args.Program, ">") {
		// A simple check is to look for > in the program string.
		// For a more precise check, we could use regex but strings.Contains is safer and catches all ">"
		// The spec says: "scan args.Program for the > redirect operator followed by a path string. If found, return EXEC_FAILED with message "AWK write redirects are not permitted"."
		// We can just use a regex like `>\s*".*"`
		re := regexp.MustCompile(`>\s*".*"`)
		if re.MatchString(args.Program) {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "AWK write redirects are not permitted", false, meta)
		}
        // Let's just block any ">" that might be a redirect. Actually, the parser might catch it or we can check the AST.
        // Wait, what if the program contains `print > "/etc/passwd"`?
        // Or `a > b`? `a > b` is not a redirect if it's a comparison.
        // The regex `>\s*".*"` specifically matches `> "/path"`. Let's also check for `>>`.
        // Let's use `>>?\s*".*"`
        re2 := regexp.MustCompile(`>>?\s*".*"`)
		if re2.MatchString(args.Program) {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "AWK write redirects are not permitted", false, meta)
		}
	}

	prog, err := parser.ParseProgram([]byte(args.Program), nil)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("AWK syntax error: %v", err), false, meta)
	}

	// Better check: walk the AST to find redirects if possible, but goawk doesn't export AST.
	// Actually, goawk interp.Config allows setting `NoExec: true` which might prevent `system()`.
	// What about file writes? `goawk` allows writing to files if we don't restrict it.
	// `NoExec: true` prevents `system()`, `NoFileIO: true` prevents file reading/writing!
	// But `NoFileIO: true` prevents reading the input file? No, it prevents `getline < "file"` and `print > "file"`.
	// We pass the input via `os.Open` to `Stdin`. So `NoFileIO` is perfect!

	file, err := os.Open(args.Path)
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrNotFound, fmt.Sprintf("failed to open file: %v", err), false, meta)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to stat file: %v", err), false, meta)
	}
	if info.IsDir() {
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Path is a directory", false, meta)
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	// Wrap file in a reader that counts lines
	cr := &countingReader{reader: file}

	config := &interp.Config{
		Stdin:    cr,
		Output:   &outBuf,
		Error:    &errBuf,
		NoExec:       true, // Block system()
		NoFileWrites: true, // Block print > "file"
		NoFileReads:  true, // Block getline < "file" (except our own Stdin)
	}

	if args.FieldSeparator != "" {
		config.Vars = []string{"FS", args.FieldSeparator}
	}

	_, err = interp.ExecProgram(prog, config)

	if err != nil {
		if strings.Contains(err.Error(), "file I/O not allowed") {
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "AWK write redirects are not permitted", false, meta)
		}

		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("AWK execution error: %v", err), false, meta)
	}

	outStr := outBuf.String()
	limits := t.policy.Limits()
	if int64(len(outStr)) > limits.MaxOutputBytes {
		outStr = outStr[:limits.MaxOutputBytes]
		meta.Truncated = true
	}

	data := phase2.AwkData{
		Output:         outStr,
		LinesProcessed: cr.lines,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}

type countingReader struct {
	reader *os.File
	lines  int
}

func (c *countingReader) Read(p []byte) (n int, err error) {
	n, err = c.reader.Read(p)
	for i := 0; i < n; i++ {
		if p[i] == '\n' {
			c.lines++
		}
	}
	return
}

func (t *awkTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("awk",
		mcp.WithDescription(t.Description()),
		mcp.WithString("program",
			mcp.Required(),
			mcp.Description("The AWK program to execute (e.g. '{print $2}')."),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("The absolute path to the input file."),
		),
		mcp.WithString("field_separator",
			mcp.Description("The field separator character (equivalent to awk -F). Default: \" \"."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := phase2.AwkArgs{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if program, ok := argsMap["program"].(string); ok {
				args.Program = program
			}
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if fs, ok := argsMap["field_separator"].(string); ok {
				args.FieldSeparator = fs
			} else {
				args.FieldSeparator = " "
			}
		}
		envelope := t.execute(ctx, args)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}
