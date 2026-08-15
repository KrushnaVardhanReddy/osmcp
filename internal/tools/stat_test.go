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

func setupStatTestEnv(t *testing.T) (string, contracts.Tool) {
	tempDir := t.TempDir()

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("data"), 0644)
	os.Mkdir(filepath.Join(tempDir, "dir1"), 0755)

	policy := &mockPolicyEngine{
		allowedRoot: tempDir,
		maxMatches:  10,
	}
	builder := response.NewBuilder()
	tool := NewStatTool(policy, builder)

	return tempDir, tool
}

func TestStatTool(t *testing.T) {
	tempDir, tool := setupStatTestEnv(t)
	stat := tool.(*statTool)

	t.Run("Stat on file", func(t *testing.T) {
		args := contracts_phase1.StatArgs{
			Path: filepath.Join(tempDir, "file.txt"),
		}
		env := stat.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.StatData)
		if data.Name != "file.txt" {
			t.Errorf("expected file.txt, got %s", data.Name)
		}
		if data.IsDir {
			t.Error("expected IsDir=false")
		}
		if data.Size != 4 { // "data"
			t.Errorf("expected size 4, got %d", data.Size)
		}
	})

	t.Run("Stat on directory", func(t *testing.T) {
		args := contracts_phase1.StatArgs{
			Path: filepath.Join(tempDir, "dir1"),
		}
		env := stat.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.StatData)
		if data.Name != "dir1" {
			t.Errorf("expected dir1, got %s", data.Name)
		}
		if !data.IsDir {
			t.Error("expected IsDir=true")
		}
	})

	t.Run("Stat missing file", func(t *testing.T) {
		args := contracts_phase1.StatArgs{
			Path: filepath.Join(tempDir, "missing"),
		}
		env := stat.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrNotFound {
			t.Errorf("expected NOT_FOUND, got %v", env.Error.Code)
		}
	})

	t.Run("Stat outside root policy denial", func(t *testing.T) {
		args := contracts_phase1.StatArgs{
			Path: "/tmp/outside_root",
		}
		env := stat.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})
}
