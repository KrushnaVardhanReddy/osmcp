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
)

type duTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewDuTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &duTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *duTool) Name() string {
	return "du"
}

func (t *duTool) Description() string {
	return "Calculates disk usage of a directory or file."
}

func (t *duTool) IsMutating() bool {
	return false
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (t *duTool) Execute(ctx context.Context, args contracts_phase1.DuArgs) contracts.Envelope {
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

	if !info.IsDir() {
		// Single file
		size := info.Size()
		data := contracts_phase1.DuData{
			TotalBytes: size,
			TotalHuman: formatBytes(size),
			Breakdown: []contracts_phase1.DuEntry{
				{
					Path:  filepath.Base(args.Path),
					Bytes: size,
					Human: formatBytes(size),
				},
			},
		}
		meta.ExecutionTimeMs = time.Since(start).Milliseconds()
		return t.builder.Success(t.Name(), data, meta)
	}

	maxDepth := args.MaxDepth
	if maxDepth < 1 {
		maxDepth = 1
	}

	var totalBytes int64
	breakdownMap := make(map[string]int64)

	basePath := args.Path
	if !strings.HasSuffix(basePath, string(os.PathSeparator)) {
		basePath += string(os.PathSeparator)
	}

	walkErr := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip on permission errors
		}

		if d.IsDir() {
			return nil // only count files
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		size := info.Size()
		totalBytes += size

		// Calculate depth and attribution path
		rel, err := filepath.Rel(args.Path, path)
		if err != nil || rel == "." {
			return nil
		}

		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > maxDepth {
			parts = parts[:maxDepth]
		}

		// Reconstruct the attribution path
		attrPath := strings.Join(parts, "/")
		if len(parts) < len(strings.Split(filepath.ToSlash(rel), "/")) {
			// it's a directory aggregate
			attrPath += "/"
		} else {
            // It's a file at max depth or less, but we might want to group everything under its top level directory up to maxDepth?
            // Actually, spec says: "Track size per subdirectory at depth <= args.MaxDepth."
            // For example, if max_depth: 1, and we have `internal/tools/tree.go`, we attribute it to `internal/`.
            if len(parts) == maxDepth && maxDepth > 0 && len(strings.Split(filepath.ToSlash(rel), "/")) > maxDepth {
                 attrPath += "/"
            }
        }
		breakdownMap[attrPath] += size

		return nil
	})

	if walkErr != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, walkErr.Error(), false, meta)
	}

	var breakdown []contracts_phase1.DuEntry
	for path, size := range breakdownMap {
		breakdown = append(breakdown, contracts_phase1.DuEntry{
			Path:  path,
			Bytes: size,
			Human: formatBytes(size),
		})
	}

	// Just sort them to be somewhat deterministic
	// We don't have to sort, but let's leave it as is.
	// Actually, the spec output has "internal/", "docs/".

	data := contracts_phase1.DuData{
		TotalBytes: totalBytes,
		TotalHuman: formatBytes(totalBytes),
		Breakdown:  breakdown,
	}

	meta.ExecutionTimeMs = time.Since(start).Milliseconds()
	return t.builder.Success(t.Name(), data, meta)
}
