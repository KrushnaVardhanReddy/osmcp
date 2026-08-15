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

// mockPolicyEngine provides a simple policy implementation for testing.
type mockPolicyEngine struct {
	allowedRoot string
	maxMatches  int
}

func (m *mockPolicyEngine) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	for _, p := range pathArgs {
		if !filepath.HasPrefix(p, m.allowedRoot) {
			return &contracts.PolicyError{Reason: "path outside allowed root"}
		}
	}
	return nil
}

func (m *mockPolicyEngine) IsToolVisible(toolName string) bool {
	return true
}

func (m *mockPolicyEngine) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{
		MaxMatches: m.maxMatches,
	}
}

func setupLsTestEnv(t *testing.T) (string, contracts.Tool) {
	tempDir := t.TempDir()

	// Create some files and directories
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, ".hidden_file"), []byte("hidden"), 0644)

	dir1 := filepath.Join(tempDir, "dir1")
	os.Mkdir(dir1, 0755)
	os.WriteFile(filepath.Join(dir1, "file2.txt"), []byte("test2"), 0644)
	os.Mkdir(filepath.Join(dir1, ".hidden_dir"), 0755)
	os.WriteFile(filepath.Join(dir1, ".hidden_dir", "file3.txt"), []byte("test3"), 0644)

	dir2 := filepath.Join(tempDir, "dir1", "dir2")
	os.Mkdir(dir2, 0755)
	os.WriteFile(filepath.Join(dir2, "file4.txt"), []byte("test4"), 0644)

	policy := &mockPolicyEngine{
		allowedRoot: tempDir,
		maxMatches:  10,
	}
	builder := response.NewBuilder()
	tool := NewLsTool(policy, builder)

	return tempDir, tool
}

func TestLsTool(t *testing.T) {
	tempDir, tool := setupLsTestEnv(t)
	ls := tool.(*lsTool)

	t.Run("INV-FILE-08: ls on file", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{Path: filepath.Join(tempDir, "file1.txt")}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got failure: %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 1 {
			t.Errorf("expected count 1, got %d", data.Count)
		}
		if data.Entries[0].Name != "file1.txt" {
			t.Errorf("expected entry name file1.txt, got %s", data.Entries[0].Name)
		}
	})

	t.Run("INV-FILE-03: Missing file", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{Path: filepath.Join(tempDir, "missing.txt")}}
		env := ls.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrNotFound {
			t.Errorf("expected NOT_FOUND, got %v", env.Error.Code)
		}
	})

	t.Run("INV-FILE-02: Policy Check Denied", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{Path: "/tmp/outside_root"}}
		env := ls.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrPolicyDenied {
			t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
		}
	})

	t.Run("Pure listing without hidden files", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{
			Path:       tempDir,
			ShowHidden: false,
			Recursive:  false,
		}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 2 { // dir1 and file1.txt
			t.Errorf("expected 2 entries, got %d", data.Count)
		}
	})

	t.Run("Listing with hidden files", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{
			Path:       tempDir,
			ShowHidden: true,
			Recursive:  false,
		}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 3 { // dir1, file1.txt, .hidden_file
			t.Errorf("expected 3 entries, got %d", data.Count)
		}
	})

	t.Run("Recursive listing depth 1", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{
			Path:       tempDir,
			ShowHidden: false,
			Recursive:  true,
			MaxDepth:   1,
		}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		// Depth 1 includes: dir1, file1.txt
		// (dir1's children are depth 2 and skipped)
		if data.Count != 2 {
			t.Errorf("expected 2 entries, got %d", data.Count)
		}
	})

	t.Run("Recursive listing max depth", func(t *testing.T) {
		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{
			Path:       tempDir,
			ShowHidden: false,
			Recursive:  true,
			MaxDepth:   3,
		}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		// Max Depth 3 should include:
		// Depth 1: dir1, file1.txt
		// Depth 2: dir1/file2.txt, dir1/dir2
		// Depth 3: dir1/dir2/file4.txt
		// (Total 5)
		if data.Count != 5 {
			t.Errorf("expected 5 entries, got %d", data.Count)
		}
	})

	t.Run("INV-FILE-05: ls Truncation", func(t *testing.T) {
		// Temporarily limit maxMatches
		ls.policy.(*mockPolicyEngine).maxMatches = 1
		defer func() { ls.policy.(*mockPolicyEngine).maxMatches = 10 }()

		args := LsArgsExtended{LsArgs: contracts_phase1.LsArgs{
			Path:       tempDir,
			ShowHidden: false,
			Recursive:  false,
		}}
		env := ls.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		if !env.Meta.Truncated {
			t.Error("expected Truncated=true")
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 1 {
			t.Errorf("expected count 1 due to truncation, got %d", data.Count)
		}
	})

}

func TestLsTool_Pattern(t *testing.T) {
	tempDir, tool := setupLsTestEnv(t)
	ls := tool.(*lsTool)

	// setupLsTestEnv creates:
	// file1.txt
	// .hidden_file
	// dir1/file2.txt
	// dir1/.hidden_dir/file3.txt
	// dir1/dir2/file4.txt

	t.Run("Pattern matching files", func(t *testing.T) {
		args := LsArgsExtended{
			LsArgs: contracts_phase1.LsArgs{Path: tempDir},
			Pattern: "*.txt",
		}
		env := ls.Execute(context.Background(), args)
		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 1 {
			t.Errorf("expected count 1, got %d", data.Count)
		}
	})

	t.Run("Pattern with recursive", func(t *testing.T) {
		args := LsArgsExtended{
			LsArgs: contracts_phase1.LsArgs{Path: tempDir, Recursive: true, MaxDepth: 2},
			Pattern: "*.txt",
		}
		env := ls.Execute(context.Background(), args)
		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.LsData)
		if data.Count != 2 {
			t.Errorf("expected count 2, got %d", data.Count)
		}
	})

	t.Run("Invalid pattern", func(t *testing.T) {
		args := LsArgsExtended{
			LsArgs: contracts_phase1.LsArgs{Path: tempDir},
			Pattern: "[",
		}
		env := ls.Execute(context.Background(), args)
		if env.OK {
			t.Fatalf("expected failure for invalid pattern")
		}
		if env.Error.Code != contracts.ErrInvalidArgs {
			t.Errorf("expected ErrInvalidArgs, got %v", env.Error.Code)
		}
	})
}

func (m *mockPolicyEngine) RunScriptConfig() contracts.RunScriptConfig {
	return contracts.RunScriptConfig{}
}

func (m *mockPolicyEngine) AllowedRoot() string {
	return "/tmp/mockroot"
}
