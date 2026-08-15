package tools

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/ulikunitz/xz"
)

type tarTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewTarTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &tarTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *tarTool) Name() string {
	return "tar"
}

func (t *tarTool) Description() string {
	return "Lists the contents of a tar archive, or extracts a single named entry from it."
}

func (t *tarTool) IsMutating() bool {
	return false
}

func (t *tarTool) Execute(ctx context.Context, args map[string]interface{}) contracts.Envelope {
	var req phase2.TarArgs
	b, err := json.Marshal(args)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "failed to parse arguments", false, contracts.Meta{})
	}
	if err := json.Unmarshal(b, &req); err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "failed to decode arguments", false, contracts.Meta{})
	}

	if req.Action != phase2.TarActionList && req.Action != phase2.TarActionExtract {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "action must be list or extract", false, contracts.Meta{})
	}
	if req.Action == phase2.TarActionExtract && req.Entry == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "entry must be provided for extract action", false, contracts.Meta{})
	}

	return t.Tar(req)
}

func (t *tarTool) Tar(req phase2.TarArgs) contracts.Envelope {
	meta := contracts.Meta{}
	ctx := context.Background()

	err := t.policy.Evaluate(ctx, t.Name(), []string{req.Path}, false)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	f, err := os.Open(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return t.builder.Failure(t.Name(), contracts.ErrNotFound, fmt.Sprintf("file not found: %s", req.Path), false, meta)
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to open file: %v", err), false, meta)
	}
	defer f.Close()

	var reader io.Reader = f
	lowerPath := strings.ToLower(req.Path)
	if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to create gzip reader: %v", err), false, meta)
		}
		defer gzr.Close()
		reader = gzr
	} else if strings.HasSuffix(lowerPath, ".tar.bz2") {
		reader = bzip2.NewReader(f)
	} else if strings.HasSuffix(lowerPath, ".tar.xz") {
		xzr, err := xz.NewReader(f)
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to create xz reader: %v", err), false, meta)
		}
		reader = xzr
	}

	tr := tar.NewReader(reader)

	if req.Action == phase2.TarActionList {
		var entries []phase2.TarEntry
		maxMatches := t.policy.Limits().MaxMatches

		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to read tar: %v", err), false, meta)
			}

			// Path traversal check
			if strings.HasPrefix(hdr.Name, "/") || strings.Contains(hdr.Name, "../") {
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "archive contains path traversal entries", false, meta)
			}

			if len(entries) >= maxMatches {
				meta.Truncated = true
				break
			}

			modeStr := hdr.FileInfo().Mode().String()
			isDir := hdr.Typeflag == tar.TypeDir || strings.HasSuffix(hdr.Name, "/")

			entries = append(entries, phase2.TarEntry{
				Name:  hdr.Name,
				Size:  hdr.Size,
				Mode:  modeStr,
				IsDir: isDir,
			})
		}

		data := phase2.TarListData{
			Entries: entries,
			Count:   len(entries),
		}
		return t.builder.Success(t.Name(), data, meta)

	} else if req.Action == phase2.TarActionExtract {
		maxOutputBytes := t.policy.Limits().MaxOutputBytes
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to read tar: %v", err), false, meta)
			}

			// Path traversal check
			if strings.HasPrefix(hdr.Name, "/") || strings.Contains(hdr.Name, "../") {
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "archive contains path traversal entries", false, meta)
			}

			if hdr.Name == req.Entry {
				if hdr.Typeflag == tar.TypeDir {
					return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "cannot extract directory as text", false, meta)
				}

				limitReader := io.LimitReader(tr, int64(maxOutputBytes)+1)
				b, err := io.ReadAll(limitReader)
				if err != nil {
					return t.builder.Failure(t.Name(), contracts.ErrExecFailed, fmt.Sprintf("failed to read entry: %v", err), false, meta)
				}

				if int64(len(b)) > maxOutputBytes {
					meta.Truncated = true
					b = b[:maxOutputBytes]
				}

				if len(b) > 0 {
					if !utf8.Valid(b) {
						return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "binary file cannot be extracted as text", false, meta)
					}
				}

				data := phase2.TarExtractData{
					Entry:   hdr.Name,
					Content: string(b),
					Size:    hdr.Size,
				}
				return t.builder.Success(t.Name(), data, meta)
			}
		}
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "entry not found in archive", false, meta)
	}

	return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "invalid action", false, meta)
}

func (t *tarTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("The absolute path to the .tar, .tar.gz, .tgz, .tar.bz2, or .tar.xz archive."),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("One of \"list\" or \"extract\"."),
		),
		mcp.WithString("entry",
			mcp.Description("The exact path of the entry inside the archive to extract. Required only when action = \"extract\"."),
		),
	)

	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid arguments")
		}

		env := t.Execute(ctx, args)
		b, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}

		content := string(b)
		res := mcp.NewToolResultText(content)
		res.IsError = !env.OK
		return res, nil
	})
}
