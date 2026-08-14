package tools

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type gitDiffTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

func NewGitDiffTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &gitDiffTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *gitDiffTool) Name() string {
	return "git_diff"
}

func (t *gitDiffTool) Description() string {
	return "Returns the unified-diff patches between two commits."
}

func (t *gitDiffTool) IsMutating() bool {
	return false
}

func (t *gitDiffTool) Execute(ctx context.Context, args contracts_phase1.GitDiffArgs) contracts.Envelope {
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

	resolveCommit := func(ref string) (*object.Commit, error) {
		hash, err := repo.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return nil, err
		}
		return repo.CommitObject(*hash)
	}

	toCommitRef := "HEAD"
	if args.ToCommit != nil && *args.ToCommit != "" {
		toCommitRef = *args.ToCommit
	}
	toCommitObj, err := resolveCommit(toCommitRef)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Failed to resolve to_commit: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	fromCommitRef := "HEAD~1"
	if args.FromCommit != nil && *args.FromCommit != "" {
		fromCommitRef = *args.FromCommit
	}
	fromCommitObj, err := resolveCommit(fromCommitRef)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "Failed to resolve from_commit: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	fromTree, err := fromCommitObj.Tree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get fromTree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	toTree, err := toCommitObj.Tree()
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to get toTree: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	patch, err := fromTree.Patch(toTree)
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrExecFailed, "Failed to compute diff: "+err.Error(), false, contracts.Meta{
			ExecutionTimeMs: time.Since(start).Milliseconds(),
		})
	}

	data := contracts_phase1.GitDiffData{
		Patches:        []contracts_phase1.GitPatch{},
		TotalAdditions: 0,
		TotalDeletions: 0,
	}

	maxOutputBytes := t.policy.Limits().MaxOutputBytes
	var currentBytes int64
	truncated := false

	for _, p := range patch.FilePatches() {
		from, to := p.Files()
		file := ""
		if to != nil {
			file = to.Path()
		} else if from != nil {
			file = from.Path()
		}

		if args.File != nil && *args.File != "" && file != *args.File {
			continue
		}

		var add, del int
		for _, chunk := range p.Chunks() {
			switch chunk.Type() {
			case diff.Add:
				add += strings.Count(chunk.Content(), "\n")
			case diff.Delete:
				del += strings.Count(chunk.Content(), "\n")
			}
		}

        // Need to create a single patch to encode it
        var b bytes.Buffer
        enc := diff.NewUnifiedEncoder(&b, diff.DefaultContextLines)

        singlePatchMock := &singleFilePatch{p}

        err := enc.Encode(singlePatchMock)
        if err != nil {
            continue
        }

		diffStr := b.String()
		diffBytes := int64(len(diffStr))

		if currentBytes+diffBytes > maxOutputBytes {
			truncated = true
			break
		}

		data.Patches = append(data.Patches, contracts_phase1.GitPatch{
			File:      file,
			Additions: add,
			Deletions: del,
			Diff:      diffStr,
		})

		data.TotalAdditions += add
		data.TotalDeletions += del
		currentBytes += diffBytes
	}

	return t.builder.Success(t.Name(), data, contracts.Meta{
		ExecutionTimeMs: time.Since(start).Milliseconds(),
		Truncated:       truncated,
	})
}

// singleFilePatch implements diff.Patch interface for a single file patch
type singleFilePatch struct {
    fp diff.FilePatch
}

func (s *singleFilePatch) FilePatches() []diff.FilePatch {
    return []diff.FilePatch{s.fp}
}

func (s *singleFilePatch) Message() string {
    return ""
}
