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

func setupTestGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	repo, err := git.PlainInit(tmpDir, false)
	assert.NoError(t, err)

	w, err := repo.Worktree()
	assert.NoError(t, err)

	file1 := filepath.Join(tmpDir, "file1.txt")
	err = os.WriteFile(file1, []byte("hello"), 0644)
	assert.NoError(t, err)

	_, err = w.Add("file1.txt")
	assert.NoError(t, err)

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err)

	return tmpDir
}

type mockGitPolicyEngine struct {
	allowed   bool
	limits    contracts.PolicyLimits
	limitsSet bool
}

func (m *mockGitPolicyEngine) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	if !m.allowed {
		return &contracts.PolicyError{Reason: "denied"}
	}
	return nil
}

func (m *mockGitPolicyEngine) IsToolVisible(toolName string) bool { return true }
func (m *mockGitPolicyEngine) Limits() contracts.PolicyLimits {
	if !m.limitsSet {
		return contracts.PolicyLimits{MaxOutputBytes: 1024, MaxMatches: 10}
	}
	return m.limits
}

func TestGitStatus_Clean(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	tool := NewGitStatusTool(policy, builder)

	args := contracts_phase1.GitStatusArgs{
		Path: repoDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitStatusArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data, ok := env.Data.(contracts_phase1.GitStatusData)
	assert.True(t, ok)
	assert.True(t, data.Clean)
	assert.Empty(t, data.Staged)
	assert.Empty(t, data.Unstaged)
	assert.Empty(t, data.Untracked)
}

func TestGitStatus_Unclean(t *testing.T) {
	repoDir := setupTestGitRepo(t)

	// Modify file
	err := os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("modified"), 0644)
	assert.NoError(t, err)

	// Add new untracked file
	err = os.WriteFile(filepath.Join(repoDir, "file2.txt"), []byte("untracked"), 0644)
	assert.NoError(t, err)

	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	tool := NewGitStatusTool(policy, builder)

	args := contracts_phase1.GitStatusArgs{
		Path: repoDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitStatusArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data, ok := env.Data.(contracts_phase1.GitStatusData)
	assert.True(t, ok)
	assert.False(t, data.Clean)

	assert.Len(t, data.Unstaged, 1)
	assert.Equal(t, "file1.txt", data.Unstaged[0].Path)
	assert.Equal(t, "M", data.Unstaged[0].Code)

	assert.Len(t, data.Untracked, 1)
	assert.Equal(t, "file2.txt", data.Untracked[0])
}

func TestGitStatus_PolicyDenied(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: false}

	tool := NewGitStatusTool(policy, builder)

	args := contracts_phase1.GitStatusArgs{
		Path: repoDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitStatusArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGitStatus_NotRepo(t *testing.T) {
	tmpDir := t.TempDir() // Empty dir, not a repo
	builder := response.NewBuilder()
	policy := &mockGitPolicyEngine{allowed: true}

	tool := NewGitStatusTool(policy, builder)

	args := contracts_phase1.GitStatusArgs{
		Path: tmpDir,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.GitStatusArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
	assert.Contains(t, env.Error.Message, "Not a git repository")
}

func (m *mockGitPolicyEngine) RunScriptConfig() contracts.RunScriptConfig {
	return contracts.RunScriptConfig{}
}

func (m *mockGitPolicyEngine) AllowedRoot() string {
	return "/tmp/mockroot"
}
