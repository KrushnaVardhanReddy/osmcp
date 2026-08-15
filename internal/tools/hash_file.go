package tools

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase4 "github.com/osmcp/osmcp/docs/contracts/phase-4"
)

type hashFileTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewHashFileTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &hashFileTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *hashFileTool) Name() string {
	return "hash_file"
}

func (t *hashFileTool) Description() string {
	return "Calculate the cryptographic hash of a file on the host filesystem."
}

func (t *hashFileTool) IsMutating() bool {
	return false
}

func (t *hashFileTool) Execute(ctx context.Context, args contracts_phase4.HashFileArgs) contracts.Envelope {
	start := time.Now()
	meta := contracts.Meta{
		ExecutionTimeMs: 0,
		Truncated:       false,
	}

	if args.Path == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "path must not be empty", false, meta)
	}

	algorithm := strings.ToLower(args.Algorithm)
	if algorithm == "" {
		algorithm = "sha256"
	}
	if algorithm != "sha256" && algorithm != "md5" && algorithm != "sha1" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "unsupported algorithm", false, meta)
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

	var hasher io.Writer
	var h interface {
		io.Writer
		Sum(b []byte) []byte
	}

	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	}
	hasher = h

	// Enforce timeout strictly
	limits := t.policy.Limits()
	timeoutMs := limits.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 10000 // 10s default
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Create a context-aware reader that cancels io.Copy if context expires
	// Alternatively, we can use a goroutine and a channel for io.Copy, but let's try a simple approach first
	// io.Copy can block. To handle timeout nicely we can do chunked reads.

	buf := make([]byte, 64*1024)
	for {
		select {
		case <-ctxWithTimeout.Done():
			meta.ExecutionTimeMs = time.Since(start).Milliseconds()
			return t.builder.Failure(t.Name(), contracts.ErrTimeout, "timeout calculating hash", false, meta)
		default:
		}

		n, err := file.Read(buf)
		if n > 0 {
			if _, werr := hasher.Write(buf[:n]); werr != nil {
				meta.ExecutionTimeMs = time.Since(start).Milliseconds()
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, werr.Error(), false, meta)
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

	hashResult := hex.EncodeToString(h.Sum(nil))

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	data := contracts_phase4.HashFileData{
		Hash:      hashResult,
		Algorithm: algorithm,
	}

	return t.builder.Success(t.Name(), data, meta)
}

func (t *hashFileTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the file to hash."),
		),
		mcp.WithString("algorithm",
			mcp.Description("Hash algorithm: sha256 (default), md5, sha1."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := contracts_phase4.HashFileArgs{
			Algorithm: "sha256",
		}

		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if alg, ok := argsMap["algorithm"].(string); ok {
				args.Algorithm = alg
			}
		}

		envelope := t.Execute(ctx, args)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}

		// Ensure proper result text handling
		res := mcp.NewToolResultText(string(resBytes))
		res.IsError = !envelope.OK
		return res, nil
	})
}
