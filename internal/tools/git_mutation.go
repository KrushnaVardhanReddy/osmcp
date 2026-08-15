package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

type gitAddTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitAddTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitAddTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitAddTool) Name() string {
	return "git_add"
}

func (t *gitAddTool) Description() string {
	return "Stages changes for the next commit."
}

func (t *gitAddTool) IsMutating() bool {
	return true
}

func (t *gitAddTool) Execute(ctx context.Context, req contracts_phase2.GitAddRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

	w, err := repo.Worktree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get worktree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	for _, p := range req.Paths {
		if p == "." {
			err = w.AddWithOptions(&git.AddOptions{All: true})
		} else {
			_, err = w.Add(p)
		}

		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to add path "+p+": "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
		}
	}

	return t.builder.Success(t.Name(), map[string]interface{}{"status": "success"}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitAddTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithArray("paths",
			mcp.Required(),
			mcp.Description("Specific paths to stage, or [\" \"] for all changes."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitAddRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if pathsObj, ok := argsMap["paths"].([]interface{}); ok {
				for _, p := range pathsObj {
					if ps, ok := p.(string); ok {
						req.Paths = append(req.Paths, ps)
					}
				}
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

type gitCommitTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitCommitTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitCommitTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitCommitTool) Name() string {
	return "git_commit"
}

func (t *gitCommitTool) Description() string {
	return "Creates a new commit containing the currently staged changes."
}

func (t *gitCommitTool) IsMutating() bool {
	return true
}

func (t *gitCommitTool) Execute(ctx context.Context, req contracts_phase2.GitCommitRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

	w, err := repo.Worktree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get worktree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	opts := &git.CommitOptions{}

	authorName := req.AuthorName
	authorEmail := req.AuthorEmail

	if authorName == "" || authorEmail == "" {
		cfg, err := repo.Config()
		if err == nil {
			if authorName == "" {
				authorName = cfg.User.Name
			}
			if authorEmail == "" {
				authorEmail = cfg.User.Email
			}
		}
	}

	if authorName != "" && authorEmail != "" {
		opts.Author = &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		}
	} else {
        return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Author name and email are required (via args or git config)", false, contracts.Meta{
            ExecutionTimeMs: time.Since(start).Milliseconds(),
        })
    }

	hash, err := w.Commit(req.Message, opts)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to commit: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	return t.builder.Success(t.Name(), map[string]interface{}{"hash": hash.String()}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitCommitTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The commit message."),
		),
		mcp.WithString("author_name",
			mcp.Description("Optional. Author name."),
		),
		mcp.WithString("author_email",
			mcp.Description("Optional. Author email."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitCommitRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if msg, ok := argsMap["message"].(string); ok {
				req.Message = msg
			}
			if an, ok := argsMap["author_name"].(string); ok {
				req.AuthorName = an
			}
			if ae, ok := argsMap["author_email"].(string); ok {
				req.AuthorEmail = ae
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

type gitCheckoutTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitCheckoutTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitCheckoutTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitCheckoutTool) Name() string {
	return "git_checkout"
}

func (t *gitCheckoutTool) Description() string {
	return "Switches to a different branch or restores files."
}

func (t *gitCheckoutTool) IsMutating() bool {
	return true
}

func (t *gitCheckoutTool) Execute(ctx context.Context, req contracts_phase2.GitCheckoutRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

	w, err := repo.Worktree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get worktree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	opts := &git.CheckoutOptions{
		Create: req.Create,
		Force:  true, // Optional: might want to let the user choose
	}

	if req.Branch != "" {
		opts.Branch = plumbing.ReferenceName("refs/heads/" + req.Branch)
		if req.Create {
			headRef, err := repo.Head()
			if err != nil {
				return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get HEAD: "+err.Error(), false, contracts.Meta{
					ExecutionTimeMs: time.Since(start).Milliseconds(),
				})
			}
			opts.Hash = headRef.Hash()
		}
	}

	err = w.Checkout(opts)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to checkout: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	return t.builder.Success(t.Name(), map[string]interface{}{"status": "success"}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitCheckoutTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithString("branch",
			mcp.Description("Optional. The name of the branch to checkout."),
		),
		mcp.WithBoolean("create",
			mcp.Description("Optional. If true, creates a new branch (git checkout -b)."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitCheckoutRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if b, ok := argsMap["branch"].(string); ok {
				req.Branch = b
			}
			if c, ok := argsMap["create"].(bool); ok {
				req.Create = c
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

type gitBranchTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitBranchTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitBranchTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitBranchTool) Name() string {
	return "git_branch"
}

func (t *gitBranchTool) Description() string {
	return "Creates or deletes branches."
}

func (t *gitBranchTool) IsMutating() bool {
	return true
}

func (t *gitBranchTool) Execute(ctx context.Context, req contracts_phase2.GitBranchRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

    if req.Action == "create" {
		headRef, err := repo.Head()
		if err != nil {
			return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get HEAD: "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
		}

		ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/" + req.BranchName), headRef.Hash())
        err = repo.Storer.SetReference(ref)
        if err != nil {
            return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to create branch: "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
        }
	} else if req.Action == "delete" {
		err := repo.Storer.RemoveReference(plumbing.ReferenceName("refs/heads/" + req.BranchName))
        if err != nil {
            return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to delete branch: "+err.Error(), false, contracts.Meta{
				ExecutionTimeMs: time.Since(start).Milliseconds(),
			})
        }
	} else {
        return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Invalid action, must be create or delete", false, contracts.Meta{
            ExecutionTimeMs: time.Since(start).Milliseconds(),
        })
    }
    return t.builder.Success(t.Name(), map[string]interface{}{"status": "success"}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitBranchTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("One of 'create', 'delete'."),
		),
		mcp.WithString("branch_name",
			mcp.Required(),
			mcp.Description("The name of the branch."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitBranchRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if a, ok := argsMap["action"].(string); ok {
				req.Action = a
			}
			if bn, ok := argsMap["branch_name"].(string); ok {
				req.BranchName = bn
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

type gitPullTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitPullTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitPullTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitPullTool) Name() string {
	return "git_pull"
}

func (t *gitPullTool) Description() string {
	return "Fetches from and integrates with another repository or a local branch."
}

func (t *gitPullTool) IsMutating() bool {
	return true
}

func (t *gitPullTool) Execute(ctx context.Context, req contracts_phase2.GitPullRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

	w, err := repo.Worktree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get worktree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	remoteName := "origin"
	if req.Remote != "" {
		remoteName = req.Remote
	}

	opts := &git.PullOptions{
		RemoteName: remoteName,
	}

	if req.Branch != "" {
		opts.ReferenceName = plumbing.ReferenceName("refs/heads/" + req.Branch)
	}

	err = w.Pull(opts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to pull: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	return t.builder.Success(t.Name(), map[string]interface{}{"status": "success"}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitPullTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithString("remote",
			mcp.Description("Optional. The remote name (default: 'origin')."),
		),
		mcp.WithString("branch",
			mcp.Description("Optional. The branch to pull."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitPullRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if r, ok := argsMap["remote"].(string); ok {
				req.Remote = r
			}
			if b, ok := argsMap["branch"].(string); ok {
				req.Branch = b
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

type gitPushTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitPushTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitPushTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitPushTool) Name() string {
	return "git_push"
}

func (t *gitPushTool) Description() string {
	return "Updates remote refs along with associated objects."
}

func (t *gitPushTool) IsMutating() bool {
	return true
}

func (t *gitPushTool) Execute(ctx context.Context, req contracts_phase2.GitPushRequest) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(req.RepoPath)
	if err != nil {
		resolvedPath = req.RepoPath
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

	remoteName := "origin"
	if req.Remote != "" {
		remoteName = req.Remote
	}

	opts := &git.PushOptions{
		RemoteName: remoteName,
		Force:      req.Force,
	}

	// In go-git, you push refs to refs using RefSpecs if you want to push specific branch
	// By default it pushes the current branch

	err = repo.Push(opts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to push: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	return t.builder.Success(t.Name(), map[string]interface{}{"status": "success"}, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}

func (t *gitPushTool) RegisterMCP(s *server.MCPServer) {
	mcpTool := mcp.NewTool(t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The root path of the Git repository."),
		),
		mcp.WithString("remote",
			mcp.Description("Optional. The remote name (default: 'origin')."),
		),
		mcp.WithString("branch",
			mcp.Description("Optional. The branch to push."),
		),
		mcp.WithBoolean("force",
			mcp.Description("Optional. If true, force push."),
		),
	)
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := contracts_phase2.GitPushRequest{}
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if ok {
			if rp, ok := argsMap["repo_path"].(string); ok {
				req.RepoPath = rp
			}
			if r, ok := argsMap["remote"].(string); ok {
				req.Remote = r
			}
			if b, ok := argsMap["branch"].(string); ok {
				req.Branch = b
			}
			if f, ok := argsMap["force"].(bool); ok {
				req.Force = f
			}
		}

		envelope := t.Execute(ctx, req)
		resBytes, err := json.Marshal(envelope)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}
