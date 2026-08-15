package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/response"
)

func setupWcTestEnv(t *testing.T) (string, contracts.Tool) {
	tempDir := t.TempDir()

	content := []byte("hello world\nthis is a test\n")
	os.WriteFile(filepath.Join(tempDir, "text.txt"), content, 0644)

	os.Mkdir(filepath.Join(tempDir, "dir1"), 0755)

	policy := &mockPolicyEngine{
		allowedRoot: tempDir,
		maxMatches:  10,
	}
	builder := response.NewBuilder()
	tool := NewWcTool(policy, builder)

	return tempDir, tool
}

func TestWcTool(t *testing.T) {
	tempDir, tool := setupWcTestEnv(t)
	wc := tool.(*wcTool)

	t.Run("Valid wc on file", func(t *testing.T) {
		args := contracts_phase1.WcArgs{
			Path: filepath.Join(tempDir, "text.txt"),
		}
		env := wc.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.WcData)
		if data.Lines != 2 {
			t.Errorf("expected 2 lines, got %d", data.Lines)
		}
		if data.Words != 6 { // hello world this is a test
			t.Errorf("expected 6 words, got %d", data.Words)
		}
		if data.Bytes != 27 {
			t.Errorf("expected 27 bytes, got %d", data.Bytes)
		}
	})

	t.Run("wc on directory", func(t *testing.T) {
		args := contracts_phase1.WcArgs{
			Path: filepath.Join(tempDir, "dir1"),
		}
		env := wc.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrInvalidArgs {
			t.Errorf("expected INVALID_ARGS, got %v", env.Error.Code)
		}
	})

	t.Run("wc missing file", func(t *testing.T) {
		args := contracts_phase1.WcArgs{
			Path: filepath.Join(tempDir, "missing.txt"),
		}
		env := wc.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrNotFound {
			t.Errorf("expected NOT_FOUND, got %v", env.Error.Code)
		}
	})

	t.Run("wc outside root policy denial", func(t *testing.T) {
		args := contracts_phase1.WcArgs{
			Path: "/tmp/outside_root",
		}
		env := wc.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})
}
