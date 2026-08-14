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

func setupCatTestEnv(t *testing.T) (string, contracts.Tool) {
	tempDir := t.TempDir()

	// Text file
	os.WriteFile(filepath.Join(tempDir, "text.txt"), []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	// Binary file
	os.WriteFile(filepath.Join(tempDir, "binary.bin"), []byte{0x00, 0x01, 0x02, 0x03, 0x04}, 0644)

	// Directory
	os.Mkdir(filepath.Join(tempDir, "dir1"), 0755)

	policy := &mockPolicyEngine{
		allowedRoot: tempDir,
		maxMatches:  10,
	}
	builder := response.NewBuilder()
	tool := NewCatTool(policy, builder)

	return tempDir, tool
}

// Ensure mockPolicyEngine implements policy limits correctly for cat.
// We override it locally in tests if needed.

type mockPolicyEngineCat struct {
	allowedRoot    string
	maxOutputBytes int64
}

func (m *mockPolicyEngineCat) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	for _, p := range pathArgs {
		if !filepath.HasPrefix(p, m.allowedRoot) {
			return &contracts.PolicyError{Reason: "path outside allowed root"}
		}
	}
	return nil
}

func (m *mockPolicyEngineCat) IsToolVisible(toolName string) bool {
	return true
}

func (m *mockPolicyEngineCat) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{
		MaxOutputBytes: m.maxOutputBytes,
	}
}

func TestCatTool(t *testing.T) {
	tempDir, tool := setupCatTestEnv(t)

	// Override policy for specific cat tests
	policy := &mockPolicyEngineCat{
		allowedRoot:    tempDir,
		maxOutputBytes: 1024,
	}
	builder := response.NewBuilder()
	cat := NewCatTool(policy, builder).(*catTool)

	t.Run("Valid full read", func(t *testing.T) {
		args := contracts_phase1.CatArgs{
			Path:      filepath.Join(tempDir, "text.txt"),
			StartLine: 1,
		}
		env := cat.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.CatData)
		if data.LinesReturned != 5 {
			t.Errorf("expected 5 lines, got %d", data.LinesReturned)
		}
		if !data.EOFReached {
			t.Error("expected EOF reached")
		}
	})

	t.Run("Valid partial read", func(t *testing.T) {
		endLine := 3
		args := contracts_phase1.CatArgs{
			Path:      filepath.Join(tempDir, "text.txt"),
			StartLine: 2,
			EndLine:   &endLine,
		}
		env := cat.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		data := env.Data.(contracts_phase1.CatData)
		if data.LinesReturned != 2 {
			t.Errorf("expected 2 lines, got %d", data.LinesReturned)
		}
		if data.EOFReached {
			t.Error("expected EOF not reached (stopped at EndLine)")
		}
		if data.Content != "line2\nline3" {
			t.Errorf("unexpected content: %q", data.Content)
		}
	})

	t.Run("INV-FILE-07: cat on Dir", func(t *testing.T) {
		args := contracts_phase1.CatArgs{
			Path: filepath.Join(tempDir, "dir1"),
		}
		env := cat.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrInvalidArgs {
			t.Errorf("expected INVALID_ARGS, got %v", env.Error.Code)
		}
	})

	t.Run("INV-FILE-04: cat Binary", func(t *testing.T) {
		args := contracts_phase1.CatArgs{
			Path: filepath.Join(tempDir, "binary.bin"),
		}
		env := cat.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrExecFailed {
			t.Errorf("expected EXEC_FAILED, got %v", env.Error.Code)
		}
		if env.Error.Message != "Cannot read binary file as text" {
			t.Errorf("unexpected message: %s", env.Error.Message)
		}
	})

	t.Run("INV-FILE-06: cat Truncation", func(t *testing.T) {
		// Restrict output heavily
		cat.policy.(*mockPolicyEngineCat).maxOutputBytes = 10

		args := contracts_phase1.CatArgs{
			Path: filepath.Join(tempDir, "text.txt"),
		}
		env := cat.Execute(context.Background(), args)

		if !env.OK {
			t.Fatalf("expected OK, got %v", env.Error)
		}
		if !env.Meta.Truncated {
			t.Error("expected Truncated=true")
		}
		data := env.Data.(contracts_phase1.CatData)
		if data.EOFReached {
			t.Error("expected eof_reached=false on truncation")
		}
		if len(data.Content) > 10 {
			t.Errorf("content exceeded max bytes: %d", len(data.Content))
		}

		// Reset policy
		cat.policy.(*mockPolicyEngineCat).maxOutputBytes = 1024
	})

	t.Run("Missing file", func(t *testing.T) {
		args := contracts_phase1.CatArgs{
			Path: filepath.Join(tempDir, "doesnotexist.txt"),
		}
		env := cat.Execute(context.Background(), args)

		if env.OK {
			t.Fatal("expected failure")
		}
		if env.Error.Code != contracts.ErrNotFound {
			t.Errorf("expected NOT_FOUND, got %v", env.Error.Code)
		}
	})

	// Keep the unused tool to avoid compiler error
	_ = tool
}
