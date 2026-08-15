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

func TestSortTool_Plain(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	os.WriteFile(filePath, []byte("c\nb\na\n"), 0644)

	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	data, ok := env.Data.(contracts_phase2.SortData)
	if !ok {
		t.Fatalf("expected SortData")
	}

	if data.Count != 3 {
		t.Errorf("expected count 3, got %d", data.Count)
	}
	if len(data.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(data.Lines))
	}
	if data.Lines[0] != "a" || data.Lines[1] != "b" || data.Lines[2] != "c" {
		t.Errorf("unexpected sort result: %v", data.Lines)
	}
}

func TestSortTool_Reverse(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	os.WriteFile(filePath, []byte("c\nb\na\n"), 0644)

	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath, Reverse: true}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	data := env.Data.(contracts_phase2.SortData)

	if data.Lines[0] != "c" || data.Lines[1] != "b" || data.Lines[2] != "a" {
		t.Errorf("unexpected reverse sort result: %v", data.Lines)
	}
}

func TestSortTool_Unique(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	// c is adjacent after sorting
	os.WriteFile(filePath, []byte("c\nb\na\nc\n"), 0644)

	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath, Unique: true}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	data := env.Data.(contracts_phase2.SortData)

	if data.Count != 3 {
		t.Errorf("expected count 3, got %d", data.Count)
	}
	if data.Lines[0] != "a" || data.Lines[1] != "b" || data.Lines[2] != "c" {
		t.Errorf("unexpected unique sort result: %v", data.Lines)
	}
}

func TestSortTool_Numeric(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	os.WriteFile(filePath, []byte("10\n2\n1\nfoo\nbar\n3\n"), 0644)

	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath, Numeric: true}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	data := env.Data.(contracts_phase2.SortData)

	// expected: 1, 2, 3, 10, bar, foo
	expected := []string{"1", "2", "3", "10", "bar", "foo"}
	for i, v := range expected {
		if data.Lines[i] != v {
			t.Errorf("at index %d expected %s, got %s", i, v, data.Lines[i])
		}
	}
}

func TestSortTool_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	os.WriteFile(filePath, []byte(""), 0644)

	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	data := env.Data.(contracts_phase2.SortData)
	if data.Count != 0 || len(data.Lines) != 0 {
		t.Errorf("expected empty lines, got count %d", data.Count)
	}
}

// mockPolicyEngineSort implements policy limits for sort testing.
type mockPolicyEngineSort struct {
	allowedRoot    string
	maxOutputBytes int64
}

func (m *mockPolicyEngineSort) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	for _, p := range pathArgs {
		if !filepath.HasPrefix(p, m.allowedRoot) {
			return &contracts.PolicyError{Reason: "path outside allowed root"}
		}
	}
	return nil
}

func (m *mockPolicyEngineSort) IsToolVisible(toolName string) bool { return true }

func (m *mockPolicyEngineSort) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{MaxOutputBytes: m.maxOutputBytes}
}

func TestSortTool_Truncation(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	os.WriteFile(filePath, []byte("aaaa\nbbbb\ncccc\n"), 0644)

	policyEngine := &mockPolicyEngineSort{
		allowedRoot:    tempDir,
		maxOutputBytes: 10,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	args := contracts_phase2.SortArgs{Path: filePath}
	env := sortTool.(contracts_phase2.SortTool).Sort(args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}
	if !env.Meta.Truncated {
		t.Errorf("expected meta.Truncated to be true")
	}
	data := env.Data.(contracts_phase2.SortData)
	if len(data.Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(data.Lines))
	}
}

func TestSortTool_Errors(t *testing.T) {
	tempDir := t.TempDir()
	policyEngine := &mockPolicyEngine{
		allowedRoot: tempDir,
	}
	builder := response.NewBuilder()
	sortTool := NewSortTool(policyEngine, builder)

	// Invalid args (empty path)
	env := sortTool.(contracts_phase2.SortTool).Sort(contracts_phase2.SortArgs{Path: ""})
	if env.OK || env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected INVALID_ARGS, got %v", env.Error)
	}

	// Policy Denied
	env = sortTool.(contracts_phase2.SortTool).Sort(contracts_phase2.SortArgs{Path: "/etc/passwd"})
	if env.OK || env.Error.Code != contracts.ErrPolicyDenied {
		t.Errorf("expected POLICY_DENIED, got %v", env.Error)
	}

	// Not Found
	env = sortTool.(contracts_phase2.SortTool).Sort(contracts_phase2.SortArgs{Path: filepath.Join(tempDir, "missing.txt")})
	if env.OK || env.Error.Code != contracts.ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %v", env.Error)
	}

	// Directory
	env = sortTool.(contracts_phase2.SortTool).Sort(contracts_phase2.SortArgs{Path: tempDir})
	if env.OK || env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected INVALID_ARGS (dir), got %v", env.Error)
	}
}
