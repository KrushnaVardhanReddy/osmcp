package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"os/exec"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
)

func writeGitMutationPolicy(t *testing.T, allowedRoot string, allowMutation bool) string {
	policyPath := filepath.Join(t.TempDir(), "mutation_policy.toml")
	allowMut := "false"
	if allowMutation {
		allowMut = "true"
	}
	err := os.WriteFile(policyPath, []byte(`
[policy]
allowed_root = "`+allowedRoot+`"
allowed_tools = ["git_add", "git_commit", "git_checkout", "git_branch", "git_pull", "git_push"]
allow_mutation = `+allowMut+`
allow_git_write = `+allowMut+`

[limits]
timeout_ms = 5000
max_output_bytes = 1048576
max_matches = 100

[audit]
destination = "stderr"
`), 0644)
	assert.NoError(t, err)
	return policyPath
}

func setupTempRepo(t *testing.T) string {
	repoDir := t.TempDir()
	_, err := git.PlainInit(repoDir, false)
	assert.NoError(t, err)

	filePath := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(filePath, []byte("init"), 0644)
	assert.NoError(t, err)

	return repoDir
}

func TestE2E_GitAdd_Success(t *testing.T) {
	repoDir := setupTempRepo(t)
	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_add"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": repoDir,
		"message":   "initial commit",
		"author_name": "Test User",
		"author_email": "test@example.com",
		"paths":     []string{"."},
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	s, _ := w.Status()
	assert.Equal(t, git.Added, s.File("test.txt").Staging)
}

func TestE2E_GitAdd_PolicyDenial(t *testing.T) {
	repoDir := setupTempRepo(t)
	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_add"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": repoDir,
		"message":   "initial commit",
		"author_name": "Test User",
		"author_email": "test@example.com",
		"paths":     []string{"."},
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}


func TestE2E_GitCommit_Success(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_commit"
	req.Params.Arguments = map[string]interface{}{
		"repo_path":    repoDir,
		"message":      "initial commit",
		"author_name":  "Test User",
		"author_email": "test@example.com",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	head, err := repo.Head()
	assert.NoError(t, err)
	assert.NotNil(t, head)
}

func TestE2E_GitCommit_PolicyDenial(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_commit"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": repoDir,
		"message":   "initial commit",
		"author_name": "Test User",
		"author_email": "test@example.com",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestE2E_GitCheckout_Success(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")
	w.Commit("initial", &git.CommitOptions{})

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_checkout"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": repoDir,
		"message":   "initial commit",
		"author_name": "Test User",
		"author_email": "test@example.com",
		"branch":    "new-branch",
		"create":    true,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	head, _ := repo.Head()
	assert.Equal(t, "refs/heads/new-branch", head.Name().String())
}

func TestE2E_GitCheckout_PolicyDenial(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")
	w.Commit("initial", &git.CommitOptions{})

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_checkout"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": repoDir,
		"message":   "initial commit",
		"author_name": "Test User",
		"author_email": "test@example.com",
		"branch":    "new-branch",
		"create":    true,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestE2E_GitBranch_Success(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")
	w.Commit("initial", &git.CommitOptions{})

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_branch"
	req.Params.Arguments = map[string]interface{}{
		"repo_path":   repoDir,
		"action":      "create",
		"branch_name": "new-branch",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.NoError(t, err)
	_, err = repo.Reference("refs/heads/new-branch", true)
	assert.NoError(t, err)
}

func TestE2E_GitBranch_PolicyDenial(t *testing.T) {
	repoDir := setupTempRepo(t)
	repo, _ := git.PlainOpen(repoDir)
	w, _ := repo.Worktree()
	w.Add(".")
	w.Commit("initial", &git.CommitOptions{})

	c := setupMCPClient(t, writeGitMutationPolicy(t, repoDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_branch"
	req.Params.Arguments = map[string]interface{}{
		"repo_path":   repoDir,
		"action":      "create",
		"branch_name": "new-branch",
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestE2E_GitPull_Success(t *testing.T) {
	remoteDir := setupTempRepo(t)
	remoteRepo, _ := git.PlainOpen(remoteDir)
	remoteW, _ := remoteRepo.Worktree()
	remoteW.Add(".")
	remoteW.Commit("initial remote", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	localDir := t.TempDir()
	localRepo, _ := git.PlainClone(localDir, false, &git.CloneOptions{
		URL: remoteDir,
	})

	// Add new commit to remote
	err := os.WriteFile(filepath.Join(remoteDir, "test2.txt"), []byte("new"), 0644)
	assert.NoError(t, err)
	remoteW.Add(".")
	remoteW.Commit("second remote", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	c := setupMCPClient(t, writeGitMutationPolicy(t, localDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_pull"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": localDir,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	localW, _ := localRepo.Worktree()
	s, _ := localW.Status()
	assert.True(t, s.IsClean())
	_, err = os.Stat(filepath.Join(localDir, "test2.txt"))
	assert.NoError(t, err)
}

func TestE2E_GitPull_PolicyDenial(t *testing.T) {
	localDir := setupTempRepo(t)

	c := setupMCPClient(t, writeGitMutationPolicy(t, localDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_pull"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": localDir,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestE2E_GitPush_Success(t *testing.T) {
	remoteDir := setupTempRepo(t)
	remoteRepo, _ := git.PlainOpen(remoteDir)
	remoteW, _ := remoteRepo.Worktree()
	remoteW.Add(".")
	remoteW.Commit("initial remote", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// To push, we need a bare remote or careful configuration
	// For testing, let's just make the remote accept pushes to current branch (or use bare)
	remoteRepo.Config() // dummy
	remoteCmd := []string{"config", "receive.denyCurrentBranch", "ignore"}
	cmd := exec.Command("git", remoteCmd...); cmd.Dir = remoteDir; cmd.Run()

	localDir := t.TempDir()
	git.PlainClone(localDir, false, &git.CloneOptions{
		URL: remoteDir,
	})

	// Add new commit to local
	err := os.WriteFile(filepath.Join(localDir, "test2.txt"), []byte("new"), 0644)
	assert.NoError(t, err)
	localRepo, _ := git.PlainOpen(localDir)
	localW, _ := localRepo.Worktree()
	localW.Add(".")
	localW.Commit("second local", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	c := setupMCPClient(t, writeGitMutationPolicy(t, localDir, true))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_push"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": localDir,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	// verify remote
	remoteHead, _ := remoteRepo.Head()
	localHead, _ := localRepo.Head()
	assert.Equal(t, localHead.Hash(), remoteHead.Hash())
}

func TestE2E_GitPush_PolicyDenial(t *testing.T) {
	localDir := setupTempRepo(t)

	c := setupMCPClient(t, writeGitMutationPolicy(t, localDir, false))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "git_push"
	req.Params.Arguments = map[string]interface{}{
		"repo_path": localDir,
	}

	res, err := c.client.CallTool(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	textRes, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	var env contracts.Envelope
	err = json.Unmarshal([]byte(textRes.Text), &env)
	assert.NoError(t, err)
	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}
