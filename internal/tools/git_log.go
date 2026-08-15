package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"encoding/json"
	"strings"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

)

type GitLogArgsExtended struct {
	contracts_phase1.GitLogArgs
	Author string `json:"author,omitempty"`
	Since  string `json:"since,omitempty"`
	Until  string `json:"until,omitempty"`
}

type gitLogTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitLogTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitLogTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitLogTool) Name() string {
	return "git_log"
}

func (t *gitLogTool) Description() string {
	return "Returns commit history from a Git repository."
}

func (t *gitLogTool) IsMutating() bool {
	return false
}

func (t *gitLogTool) Execute(ctx context.Context, args GitLogArgsExtended) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(args.Path)
	if err != nil {
		resolvedPath = args.Path
	}

	if err := t.policy.Evaluate(ctx, t.Name(), []string{resolvedPath}, t.IsMutating()); err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	repo, err := git.PlainOpenWithOptions(resolvedPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Not a git repository: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	opts := &git.LogOptions{}

	if args.Branch != nil && *args.Branch != "" {
		hash, err := repo.ResolveRevision(plumbing.Revision(*args.Branch))
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Failed to resolve branch: "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
		}
		opts.From = *hash
	}

	if args.File != nil && *args.File != "" {
		opts.FileName = args.File
	}

	if args.Since != "" {
		sinceTime, err := time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "invalid since date format (must be RFC3339): "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
		}
		opts.Since = &sinceTime
	}

	if args.Until != "" {
		untilTime, err := time.Parse(time.RFC3339, args.Until)
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "invalid until date format (must be RFC3339): "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
		}
		opts.Until = &untilTime
	}

	cIter, err := repo.Log(opts)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get git log: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	data := contracts_phase1.GitLogData{
		Commits: []contracts_phase1.GitCommit{},
		Count:   0,
	}

	maxCommits := args.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 20
	} else if maxCommits > 200 {
		maxCommits = 200
	}

	policyMax := t.policy.Limits().MaxMatches
	if maxCommits > policyMax {
		maxCommits = policyMax
	}

	truncated := false

	authorLower := ""
	if args.Author != "" {
		authorLower = strings.ToLower(args.Author)
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		if authorLower != "" {
			if !strings.Contains(strings.ToLower(c.Author.Name), authorLower) && !strings.Contains(strings.ToLower(c.Author.Email), authorLower) {
				return nil
			}
		}

		if data.Count >= maxCommits {
			truncated = true
			return context.Canceled
		}

		data.Commits = append(data.Commits, contracts_phase1.GitCommit{
			Hash:      c.Hash.String(),
			ShortHash: c.Hash.String()[:7],
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Date:      c.Author.When,
			Message:   c.Message,
		})
		data.Count++
		return nil
	})

	// context.Canceled is expected if we hit max commits
	if err != nil && err != context.Canceled {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed iterating commits: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

    // According to contract spec, if there were more commits than maxCommits set meta.Truncated = true
    // we already checked via the ForEach and set it appropriately

	return t.builder.Success(t.Name(), data, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
		Truncated:       truncated,
	})
}

func (t *gitLogTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool("git_log",
		mcp.WithDescription(t.Description()),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the repository root."),
		),
		mcp.WithNumber("max_commits",
			mcp.Description("Maximum number of commits to return. (default: 20)"),
		),
		mcp.WithString("file",
			mcp.Description("Optional. If set, only return commits that touched this file path."),
		),
		mcp.WithString("branch",
			mcp.Description("Optional. Branch to read history from (defaults to current HEAD)."),
		),
		mcp.WithString("author",
			mcp.Description("Optional. Filter commits by author name or email (case-insensitive substring)."),
		),
		mcp.WithString("since",
			mcp.Description("Optional. Filter commits after this date (RFC3339, e.g. \"2024-01-01T00:00:00Z\")."),
		),
		mcp.WithString("until",
			mcp.Description("Optional. Filter commits before this date (RFC3339)."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := GitLogArgsExtended{
			GitLogArgs: contracts_phase1.GitLogArgs{
				MaxCommits: 20,
			},
		}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if path, ok := argsMap["path"].(string); ok {
				args.Path = path
			}
			if maxCommits, ok := argsMap["max_commits"].(float64); ok {
				args.MaxCommits = int(maxCommits)
			}
			if file, ok := argsMap["file"].(string); ok {
				args.File = &file
			}
			if branch, ok := argsMap["branch"].(string); ok {
				args.Branch = &branch
			}
			if author, ok := argsMap["author"].(string); ok {
				args.Author = author
			}
			if since, ok := argsMap["since"].(string); ok {
				args.Since = since
			}
			if until, ok := argsMap["until"].(string); ok {
				args.Until = until
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
