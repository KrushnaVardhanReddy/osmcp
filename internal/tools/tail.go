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

type tailTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewTailTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &tailTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *tailTool) Name() string {
	return "tail"
}

func (t *tailTool) Description() string {
	return "Returns the last N lines of a file."
}

func (t *tailTool) IsMutating() bool {
	return false
}

func (t *tailTool) Execute(ctx context.Context, args contracts_phase1.TailArgs) contracts.Envelope {
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
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "cannot run tail on a directory", false, meta)
	}

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

	contentType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(contentType, "text/") {
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

	_, err = file.Seek(0, 0)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to seek: %v", err), false, meta)
	}

	linesLimit := args.Lines
	if linesLimit <= 0 {
		linesLimit = 10
	}

	// Ring buffer approach to avoid loading full file into memory
	buf := make([]string, linesLimit)
	idx := 0
	count := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buf[idx%linesLimit] = scanner.Text()
		idx++
		count++
	}

	if err := scanner.Err(); err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("scanner error: %v", err), false, meta)
	}

	var sb strings.Builder
	var linesReturned int

	if count < linesLimit {
		// We read fewer lines than the limit, so everything is in buf[0..count-1]
		for i := 0; i < count; i++ {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(buf[i])
		}
		linesReturned = count
	} else {
		// We read >= linesLimit lines, so we need to reconstruct from buf[(idx)%linesLimit] onwards
		startIdx := idx % linesLimit
		for i := 0; i < linesLimit; i++ {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(buf[(startIdx+i)%linesLimit])
		}
		linesReturned = linesLimit
	}

	data := contracts_phase1.TailData{
		Content:       sb.String(),
		LinesReturned: linesReturned,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}
