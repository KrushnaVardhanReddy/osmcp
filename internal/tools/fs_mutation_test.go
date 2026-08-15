package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/osmcp/osmcp/internal/response"
)

type mockPolicyEngineFs struct {
	allowedRoot   string
	allowMutation bool
}

func (m *mockPolicyEngineFs) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	if isMutating && !m.allowMutation {
		return &contracts.PolicyError{Reason: "mutation not permitted"}
	}
	for _, p := range pathArgs {
		if !filepath.HasPrefix(p, m.allowedRoot) {
			return &contracts.PolicyError{Reason: "path outside allowed root"}
		}
	}
	return nil
}

func (m *mockPolicyEngineFs) IsToolVisible(toolName string) bool {
	return true
}

func (m *mockPolicyEngineFs) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{}
}

func setupFsTestEnv(t *testing.T) (string, contracts.PolicyEngine, contracts.EnvelopeBuilder) {
	tempDir := t.TempDir()
	policy := &mockPolicyEngineFs{
		allowedRoot:   tempDir,
		allowMutation: true,
	}
	builder := response.NewBuilder()
	return tempDir, policy, builder
}

func TestWriteFileTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewWriteFileTool(policy, builder).(*writeFileTool)

	t.Run("Valid write new file", func(t *testing.T) {
		args := contracts_phase2.WriteFileRequest{
			Path:    filepath.Join(tempDir, "new.txt"),
			Content: "hello",
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		content, _ := os.ReadFile(args.Path)
		if string(content) != "hello" {
			t.Errorf("expected 'hello', got %s", string(content))
		}
	})

	t.Run("Overwrite existing without flag", func(t *testing.T) {
		args := contracts_phase2.WriteFileRequest{
			Path:      filepath.Join(tempDir, "new.txt"),
			Content:   "world",
			Overwrite: false,
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrInvalidArgs {
			t.Errorf("expected INVALID_ARGS, got %v", env.Error.Code)
		}
	})

	t.Run("Overwrite existing with flag", func(t *testing.T) {
		args := contracts_phase2.WriteFileRequest{
			Path:      filepath.Join(tempDir, "new.txt"),
			Content:   "world",
			Overwrite: true,
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		content, _ := os.ReadFile(args.Path)
		if string(content) != "world" {
			t.Errorf("expected 'world', got %s", string(content))
		}
	})

	t.Run("Policy mutation denied", func(t *testing.T) {
		policy.(*mockPolicyEngineFs).allowMutation = false
		args := contracts_phase2.WriteFileRequest{
			Path:    filepath.Join(tempDir, "denied.txt"),
			Content: "no",
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
		policy.(*mockPolicyEngineFs).allowMutation = true
	})
}

func TestAppendFileTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewAppendFileTool(policy, builder).(*appendFileTool)

	t.Run("Valid append creates file", func(t *testing.T) {
		args := contracts_phase2.AppendFileRequest{
			Path:    filepath.Join(tempDir, "append.txt"),
			Content: "hello",
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		content, _ := os.ReadFile(args.Path)
		if string(content) != "hello" {
			t.Errorf("expected 'hello', got %s", string(content))
		}
	})

	t.Run("Valid append existing file", func(t *testing.T) {
		args := contracts_phase2.AppendFileRequest{
			Path:    filepath.Join(tempDir, "append.txt"),
			Content: " world",
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		content, _ := os.ReadFile(args.Path)
		if string(content) != "hello world" {
			t.Errorf("expected 'hello world', got %s", string(content))
		}
	})
}

func TestMkdirTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewMkdirTool(policy, builder).(*mkdirTool)

	t.Run("Valid mkdir", func(t *testing.T) {
		args := contracts_phase2.MkdirRequest{
			Path: filepath.Join(tempDir, "newdir", "subdir"),
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		info, err := os.Stat(args.Path)
		if err != nil || !info.IsDir() {
			t.Errorf("expected directory to exist")
		}
	})

	t.Run("Policy denied outside root", func(t *testing.T) {
		args := contracts_phase2.MkdirRequest{
			Path: "/tmp/outside_root_dir",
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})
}

func TestRmTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewRmTool(policy, builder).(*rmTool)

	// Setup some files and dirs
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("data"), 0644)
	os.Mkdir(filepath.Join(tempDir, "dir1"), 0755)
	os.WriteFile(filepath.Join(tempDir, "dir1", "file2.txt"), []byte("data"), 0644)

	t.Run("Valid rm file", func(t *testing.T) {
		args := contracts_phase2.RmRequest{
			Path: filepath.Join(tempDir, "file1.txt"),
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		if _, err := os.Stat(args.Path); !os.IsNotExist(err) {
			t.Errorf("expected file to be removed")
		}
	})

	t.Run("Rm dir without recursive fails", func(t *testing.T) {
		args := contracts_phase2.RmRequest{
			Path:      filepath.Join(tempDir, "dir1"),
			Recursive: false,
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrInvalidArgs {
			t.Errorf("expected INVALID_ARGS, got %v", env.Error.Code)
		}
	})

	t.Run("Rm dir with recursive succeeds", func(t *testing.T) {
		args := contracts_phase2.RmRequest{
			Path:      filepath.Join(tempDir, "dir1"),
			Recursive: true,
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		if _, err := os.Stat(args.Path); !os.IsNotExist(err) {
			t.Errorf("expected dir to be removed")
		}
	})
}

func TestMvTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewMvTool(policy, builder).(*mvTool)

	os.WriteFile(filepath.Join(tempDir, "src.txt"), []byte("data"), 0644)

	t.Run("Valid move file", func(t *testing.T) {
		args := contracts_phase2.MvRequest{
			Source:      filepath.Join(tempDir, "src.txt"),
			Destination: filepath.Join(tempDir, "dst.txt"),
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		if _, err := os.Stat(args.Source); !os.IsNotExist(err) {
			t.Errorf("expected source to not exist")
		}
		if _, err := os.Stat(args.Destination); os.IsNotExist(err) {
			t.Errorf("expected destination to exist")
		}
	})

	t.Run("Policy denied on source", func(t *testing.T) {
		args := contracts_phase2.MvRequest{
			Source:      "/tmp/outside_root.txt",
			Destination: filepath.Join(tempDir, "dst2.txt"),
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})

	t.Run("Policy denied on destination", func(t *testing.T) {
		args := contracts_phase2.MvRequest{
			Source:      filepath.Join(tempDir, "dst.txt"),
			Destination: "/tmp/outside_root.txt",
		}
		env := tool.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})
}

func TestCpTool(t *testing.T) {
	tempDir, policy, builder := setupFsTestEnv(t)
	tool := NewCpTool(policy, builder).(*cpTool)

	os.Mkdir(filepath.Join(tempDir, "srcdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "srcdir", "file1.txt"), []byte("data1"), 0644)

	t.Run("Valid recursive copy", func(t *testing.T) {
		args := contracts_phase2.CpRequest{
			Source:      filepath.Join(tempDir, "srcdir"),
			Destination: filepath.Join(tempDir, "dstdir"),
		}
		env := tool.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}

		content, err := os.ReadFile(filepath.Join(args.Destination, "file1.txt"))
		if err != nil {
			t.Fatalf("expected copied file to exist")
		}
		if string(content) != "data1" {
			t.Errorf("expected 'data1', got %s", string(content))
		}
	})
}
