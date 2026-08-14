package tools

import (
	"context"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type gitStatusTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitStatusTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitStatusTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitStatusTool) Name() string {
	return "git_status"
}

func (t *gitStatusTool) Description() string {
	return "Inspects the working tree status of a Git repository."
}

func (t *gitStatusTool) IsMutating() bool {
	return false
}

func (t *gitStatusTool) Execute(ctx context.Context, args contracts_phase1.GitStatusArgs) contracts.Envelope {
	start := time.Now()

	resolvedPath, err := filepath.EvalSymlinks(args.Path)
	if err != nil {
		// if path does not exist or invalid symlink, try using original path
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

	head, err := repo.Head()
	var branch, headCommit string
	if err == nil && head != nil {
		branch = head.Name().Short()
		headCommit = head.Hash().String()[:7]
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get worktree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	status, err := worktree.Status()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get status: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	data := contracts_phase1.GitStatusData{
		Branch:     branch,
		HeadCommit: headCommit,
		Clean:      status.IsClean(),
		Staged:     []contracts_phase1.GitFileStatus{},
		Unstaged:   []contracts_phase1.GitFileStatus{},
		Untracked:  []string{},
	}

	for p, s := range status {
		if s.Staging != git.Unmodified && s.Staging != git.Untracked {
			data.Staged = append(data.Staged, contracts_phase1.GitFileStatus{
				Path: p,
				Code: string(s.Staging),
			})
		}
		if s.Worktree != git.Unmodified && s.Worktree != git.Untracked {
			data.Unstaged = append(data.Unstaged, contracts_phase1.GitFileStatus{
				Path: p,
				Code: string(s.Worktree),
			})
		}
		if s.Staging == git.Untracked || s.Worktree == git.Untracked {
			data.Untracked = append(data.Untracked, p)
		}
	}

	return t.builder.Success(t.Name(), data, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	})
}
