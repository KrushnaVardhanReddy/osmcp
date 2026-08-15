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

func TestGitLog_Success(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	// Add second commit
	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file2.txt"), []byte("second"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("file2.txt")
	assert.NoError(t, err)

	_, err = w.Commit("Second commit msg", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test2",
			Email: "test2@example.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err)

	tool := NewGitLogTool(policy, builder)

	args := GitLogArgsExtended{
		GitLogArgs: contracts_phase1.GitLogArgs{
			Path:       repoDir,
			MaxCommits: 10,
		},
	}

	env := tool.(interface {
		Execute(context.Context, GitLogArgsExtended) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data, ok := env.Data.(contracts_phase1.GitLogData)
	assert.True(t, ok)

	assert.Equal(t, 2, data.Count)
	assert.Len(t, data.Commits, 2)
	assert.Equal(t, "Second commit msg", data.Commits[0].Message)
	assert.Equal(t, "Initial commit", data.Commits[1].Message)
}

func TestGitLog_Truncation(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()

	// Set policy limit to 1
	policy := &mockGitPolicyEngine{
		allowed:   true,
		limits:    contracts.PolicyLimits{MaxMatches: 1}, // Cap at 1
		limitsSet: true,
	}

	// Make sure we have 2 commits
	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file2.txt"), []byte("second"), 0644)
	assert.NoError(t, err)
	_, err = w.Add("file2.txt")
	assert.NoError(t, err)
	_, err = w.Commit("Second", &git.CommitOptions{Author: &object.Signature{Name: "Test", When: time.Now()}})
	assert.NoError(t, err)

	tool := NewGitLogTool(policy, builder)

	args := GitLogArgsExtended{
		GitLogArgs: contracts_phase1.GitLogArgs{
			Path:       repoDir,
			MaxCommits: 10, // Request 10, but policy limits to 1
		},
	}

	env := tool.(interface {
		Execute(context.Context, GitLogArgsExtended) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	assert.True(t, env.Meta.Truncated)
	data := env.Data.(contracts_phase1.GitLogData)
	assert.Equal(t, 1, data.Count)
}

func TestGitLog_FileFilter(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file2.txt"), []byte("second"), 0644)
	assert.NoError(t, err)
	_, err = w.Add("file2.txt")
	assert.NoError(t, err)
	_, err = w.Commit("Second commit msg", &git.CommitOptions{Author: &object.Signature{Name: "Test", When: time.Now()}})
	assert.NoError(t, err)

	tool := NewGitLogTool(policy, builder)

	fileTarget := "file2.txt"
	args := GitLogArgsExtended{
		GitLogArgs: contracts_phase1.GitLogArgs{
			Path:       repoDir,
			MaxCommits: 10,
			File:       &fileTarget,
		},
	}

	env := tool.(interface {
		Execute(context.Context, GitLogArgsExtended) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GitLogData)

	// Should only match the second commit which touches file2.txt
	assert.Equal(t, 1, data.Count)
	assert.Equal(t, "Second commit msg", data.Commits[0].Message)
}

func TestGitLog_Filters(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	repo, err := git.PlainOpen(repoDir)
	assert.NoError(t, err)
	w, err := repo.Worktree()
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file3.txt"), []byte("content"), 0644)
	assert.NoError(t, err)
	_, err = w.Add("file3.txt")
	assert.NoError(t, err)

	// Create commits with specific authors and dates
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err = w.Commit("Commit 1", &git.CommitOptions{Author: &object.Signature{Name: "Alice", Email: "alice@example.com", When: t1}})
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "file4.txt"), []byte("content2"), 0644)
	assert.NoError(t, err)
	_, err = w.Add("file4.txt")
	assert.NoError(t, err)

	t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = w.Commit("Commit 2", &git.CommitOptions{Author: &object.Signature{Name: "Bob", Email: "bob@example.com", When: t2}})
	assert.NoError(t, err)

	tool := NewGitLogTool(policy, builder)

	t.Run("Author filter", func(t *testing.T) {
		args := GitLogArgsExtended{
			GitLogArgs: contracts_phase1.GitLogArgs{Path: repoDir, MaxCommits: 10},
			Author: "Alice",
		}
		env := tool.(interface {
			Execute(context.Context, GitLogArgsExtended) contracts.Envelope
		}).Execute(context.Background(), args)

		assert.True(t, env.OK)
		data := env.Data.(contracts_phase1.GitLogData)
		// Assuming setupGitTestRepo already has some commits, but "Alice" is only 1.
		count := 0
		for _, c := range data.Commits {
			if c.Author == "Alice" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("Since filter", func(t *testing.T) {
		args := GitLogArgsExtended{
			GitLogArgs: contracts_phase1.GitLogArgs{Path: repoDir, MaxCommits: 10},
			Since: "2023-06-01T00:00:00Z",
		}
		env := tool.(interface {
			Execute(context.Context, GitLogArgsExtended) contracts.Envelope
		}).Execute(context.Background(), args)

		assert.True(t, env.OK)
		data := env.Data.(contracts_phase1.GitLogData)
		// Should only find Bob's commit (2024)
		countBob := 0
		countAlice := 0
		for _, c := range data.Commits {
			if c.Author == "Bob" {
				countBob++
			}
			if c.Author == "Alice" {
				countAlice++
			}
		}
		assert.Equal(t, 1, countBob)
		assert.Equal(t, 0, countAlice)
	})

	t.Run("Until filter", func(t *testing.T) {
		args := GitLogArgsExtended{
			GitLogArgs: contracts_phase1.GitLogArgs{Path: repoDir, MaxCommits: 10},
			Until: "2023-06-01T00:00:00Z",
		}
		env := tool.(interface {
			Execute(context.Context, GitLogArgsExtended) contracts.Envelope
		}).Execute(context.Background(), args)

		assert.True(t, env.OK)
		data := env.Data.(contracts_phase1.GitLogData)
		countAlice := 0
		countBob := 0
		for _, c := range data.Commits {
			if c.Author == "Alice" {
				countAlice++
			}
			if c.Author == "Bob" {
				countBob++
			}
		}
		assert.Equal(t, 1, countAlice)
		assert.Equal(t, 0, countBob)
	})

	t.Run("Invalid date filter", func(t *testing.T) {
		args := GitLogArgsExtended{
			GitLogArgs: contracts_phase1.GitLogArgs{Path: repoDir, MaxCommits: 10},
			Since: "not-a-date",
		}
		env := tool.(interface {
			Execute(context.Context, GitLogArgsExtended) contracts.Envelope
		}).Execute(context.Background(), args)

		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
	})
}
