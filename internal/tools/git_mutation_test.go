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
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

// Helper to create a temp repo with an initial commit
func setupGitRepo(t *testing.T) (string, *git.Repository) {
	tmpDir := t.TempDir()
	repo, err := git.PlainInit(tmpDir, false)
	assert.NoError(t, err)

	file := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(file, []byte("Hello, Git"), 0644)
	assert.NoError(t, err)

	w, err := repo.Worktree()
	assert.NoError(t, err)
	_, err = w.Add("README.md")
	assert.NoError(t, err)
	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	assert.NoError(t, err)

	return tmpDir, repo
}

// Dummy logger for tests
type dummyAuditLogger struct{}

func (l *dummyAuditLogger) Log(entry contracts.AuditEntry) {}

func setupMutationPolicyEngine(t *testing.T, allowedRoot string, allowMutation bool) contracts.PolicyEngine {
	p := policy.DefaultPolicy()
	p.PolicyConfig.AllowedRoot = allowedRoot
	p.PolicyConfig.AllowMutation = allowMutation
    p.PolicyConfig.AllowGitWrite = allowMutation
	p.PolicyConfig.AllowedTools = append(p.PolicyConfig.AllowedTools, "git_add", "git_commit", "git_checkout", "git_branch", "git_pull", "git_push")
	return policy.NewEngine(p, &dummyAuditLogger{})
}

func TestGitAddTool(t *testing.T) {
	repoDir, _ := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitAddTool(policyEngine, builder)

	err := os.WriteFile(filepath.Join(repoDir, "new_file.txt"), []byte("data"), 0644)
	assert.NoError(t, err)

	ctx := context.Background()

	// Test 1: Successful Add
	req := contracts_phase2.GitAddRequest{
		RepoPath: repoDir,
		Paths:    []string{"new_file.txt"},
	}
	// Call execute directly as GitMutationTool does not have GitAdd since its not typed there
	env := tool.(*gitAddTool).Execute(ctx, req)
	assert.True(t, env.OK)
	assert.Nil(t, env.Error)

	// Test 2: Policy Denied (Mutation)
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitAddTool(readOnlyPolicy, builder)
	env = readOnlyTool.(*gitAddTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGitCommitTool(t *testing.T) {
	repoDir, repo := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitCommitTool(policyEngine, builder)

	w, _ := repo.Worktree()
	err := os.WriteFile(filepath.Join(repoDir, "new_file.txt"), []byte("data"), 0644)
	assert.NoError(t, err)
	w.Add("new_file.txt")

	ctx := context.Background()

	// Test 1: Successful Commit
	req := contracts_phase2.GitCommitRequest{
		RepoPath:    repoDir,
		Message:     "Add new file",
		AuthorName:  "Test User",
		AuthorEmail: "test@example.com",
	}
	env := tool.(*gitCommitTool).Execute(ctx, req)
	assert.True(t, env.OK)
	assert.Nil(t, env.Error)
    assert.NotNil(t, env.Data)
    dataMap := env.Data.(map[string]interface{})
    assert.NotEmpty(t, dataMap["hash"])

	// Test 2: Policy Denied (Mutation)
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitCommitTool(readOnlyPolicy, builder)
	env = readOnlyTool.(*gitCommitTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGitCheckoutTool(t *testing.T) {
	repoDir, _ := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitCheckoutTool(policyEngine, builder)

	ctx := context.Background()

	// Test 1: Successful Checkout new branch
	req := contracts_phase2.GitCheckoutRequest{
		RepoPath: repoDir,
		Branch:   "new-feature",
		Create:   true,
	}
	env := tool.(*gitCheckoutTool).Execute(ctx, req)
	assert.True(t, env.OK)
	assert.Nil(t, env.Error)

	// Test 2: Policy Denied (Mutation)
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitCheckoutTool(readOnlyPolicy, builder)
	env = readOnlyTool.(*gitCheckoutTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGitBranchTool(t *testing.T) {
	repoDir, _ := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitBranchTool(policyEngine, builder)

	ctx := context.Background()

	// Test 1: Successful create branch
	req := contracts_phase2.GitBranchRequest{
		RepoPath:   repoDir,
		Action:     "create",
		BranchName: "test-branch",
	}
	env := tool.(*gitBranchTool).Execute(ctx, req)
	assert.True(t, env.OK)
	assert.Nil(t, env.Error)

	// Test 2: Policy Denied (Mutation)
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitBranchTool(readOnlyPolicy, builder)
	env = readOnlyTool.(*gitBranchTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGitPullTool(t *testing.T) {
	repoDir, _ := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitPullTool(policyEngine, builder)

	ctx := context.Background()

	// Test 1: Policy Denied (Mutation)
	req := contracts_phase2.GitPullRequest{
		RepoPath: repoDir,
		Remote:   "origin",
		Branch:   "main",
	}
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitPullTool(readOnlyPolicy, builder)
	env := readOnlyTool.(*gitPullTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)

    // Note: We don't have a real remote to test successful pulling against without a lot of setup
    // But testing execution boundaries is what really matters here for these wrapper tools.
    env = tool.(*gitPullTool).Execute(ctx, req)
	assert.False(t, env.OK) // Fails due to no remote
}

func TestGitPushTool(t *testing.T) {
	repoDir, _ := setupGitRepo(t)
	builder := response.NewBuilder()
	policyEngine := setupMutationPolicyEngine(t, repoDir, true)
	tool := NewGitPushTool(policyEngine, builder)

	ctx := context.Background()

	// Test 1: Policy Denied (Mutation)
	req := contracts_phase2.GitPushRequest{
		RepoPath: repoDir,
		Remote:   "origin",
		Branch:   "main",
	}
	readOnlyPolicy := setupMutationPolicyEngine(t, repoDir, false)
	readOnlyTool := NewGitPushTool(readOnlyPolicy, builder)
	env := readOnlyTool.(*gitPushTool).Execute(ctx, req)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)

    env = tool.(*gitPushTool).Execute(ctx, req)
	assert.False(t, env.OK) // Fails due to no remote
}
