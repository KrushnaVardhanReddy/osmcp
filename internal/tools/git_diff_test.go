package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func TestGitDiff_Success(t *testing.T) {
	repoDir := setupTestGitRepo(t) // From git_status_test.go
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	// Make a second commit to diff against
	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("hello\nworld"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("file1.txt")
	assert.NoError(t, err)

	_, err = w.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err)

	tool := NewGitDiffTool(policy, builder)

	args := contracts_phase1.GitDiffArgs{
		Path: repoDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitDiffArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data, ok := env.Data.(contracts_phase1.GitDiffData)
	assert.True(t, ok)

	assert.Len(t, data.Patches, 1)
	assert.Equal(t, "file1.txt", data.Patches[0].File)
	assert.Equal(t, 1, data.Patches[0].Additions)
	assert.Equal(t, 0, data.Patches[0].Deletions) // hello -> hello\nworld is 1 add
	assert.Contains(t, data.Patches[0].Diff, "+world")
}

func TestGitDiff_InvalidCommit(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	tool := NewGitDiffTool(policy, builder)

	badCommit := "invalidhash"
	args := contracts_phase1.GitDiffArgs{
		Path:       repoDir,
		FromCommit: &badCommit,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitDiffArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestGitDiff_Truncation(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	// Set very small max bytes
	policy := &mockGitPolicyEngine{
		allowed:   true,
		limits:    contracts.PolicyLimits{MaxOutputBytes: 10, MaxMatches: 10},
		limitsSet: true,
	}

	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("a very long change that exceeds ten bytes easily"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("file1.txt")
	assert.NoError(t, err)

	_, err = w.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err)

	tool := NewGitDiffTool(policy, builder)

	args := contracts_phase1.GitDiffArgs{
		Path: repoDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitDiffArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	assert.True(t, env.Meta.Truncated)
}
