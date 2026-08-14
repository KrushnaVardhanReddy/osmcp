package tools

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type headTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewHeadTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &headTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *headTool) Name() string {
	return "head"
}

func (t *headTool) Description() string {
	return "Returns the first N lines of a file."
}

func (t *headTool) IsMutating() bool {
	return false
}

func (t *headTool) Execute(ctx context.Context, args contracts_phase1.HeadArgs) contracts.Envelope {
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

	if info.IsDir() {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "cannot run head on a directory", false, meta)
	}

	// Detect binary
	file, err := os.Open(args.Path)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to open file: %v", err), false, meta)
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err.Error() != "EOF" {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to read file header: %v", err), false, meta)
	}

	// Check if binary
	contentType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(contentType, "text/") {
		// http.DetectContentType is sometimes too strict (e.g. JSON might not be text/).
		// Let's do a simple null byte check to be safe and adhere to "same check as cat" ->
		// "detect binary files (e.g., by checking if the first 512 bytes contain null bytes or using http.DetectContentType)"
		hasNull := false
		for _, b := range header[:n] {
			if b == 0 {
				hasNull = true
				break
			}
		}
		if hasNull || contentType == "application/octet-stream" {
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Cannot read binary file", false, meta)
		}
	}

	// Reset file pointer
	_, err = file.Seek(0, 0)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to seek: %v", err), false, meta)
	}

	scanner := bufio.NewScanner(file)
	var sb strings.Builder
	linesRead := 0
	eofReached := true
	linesLimit := args.Lines
	if linesLimit <= 0 {
		linesLimit = 10
	}

	for scanner.Scan() {
		if linesRead >= linesLimit {
			eofReached = false
			break
		}
		if linesRead > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(scanner.Text())
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("scanner error: %v", err), false, meta)
	}

	data := contracts_phase1.HeadData{
		Content:       sb.String(),
		LinesReturned: linesRead,
		EOFReached:    eofReached,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}
