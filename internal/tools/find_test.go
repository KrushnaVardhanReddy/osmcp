package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
)

type mockPolicyEngine struct {
	visibleTools []string
	limits       contracts.PolicyLimits
	evalErr      error
}

func (m *mockPolicyEngine) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	return m.evalErr
}

func (m *mockPolicyEngine) IsToolVisible(toolName string) bool {
	for _, t := range m.visibleTools {
		if t == toolName {
			return true
		}
	}
	return false
}

func (m *mockPolicyEngine) Limits() contracts.PolicyLimits {
	return m.limits
}

type mockEnvelopeBuilder struct{}

func (m *mockEnvelopeBuilder) Success(tool string, data interface{}, meta contracts.Meta) contracts.Envelope {
	return contracts.Envelope{
		Version: "1",
		OK:      true,
		Tool:    tool,
		Data:    data,
		Meta:    meta,
	}
}

func (m *mockEnvelopeBuilder) Failure(tool string, code contracts.ErrorCode, message string, retryable bool, meta contracts.Meta) contracts.Envelope {
	return contracts.Envelope{
		Version: "1",
		OK:      false,
		Tool:    tool,
		Error: &contracts.Error{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
		Meta: meta,
	}
}

func setupTempDir(t *testing.T) string {
	dir := t.TempDir()

	// Create some files and directories
	os.MkdirAll(filepath.Join(dir, "sub", "sub2"), 0755)

	// Create regular files
	files := []string{
		"file1.txt",
		"file2.go",
		filepath.Join("sub", "file3.go"),
		filepath.Join("sub", "sub2", "file4.txt"),
	}

	for _, f := range files {
		fullPath := filepath.Join(dir, f)
		err := os.WriteFile(fullPath, []byte("hello"), 0644)
		if err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
	}

	// Create symlink for testing (INV-FIND-07)
	err := os.Symlink(filepath.Join(dir, "sub", "sub2"), filepath.Join(dir, "symlink_dir"))
	if err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	return dir
}

// INV-FIND-01: Pure Go
func TestFindTool_BasicExecution(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
	}, &mockEnvelopeBuilder{})

	nameGlob := "*.go"
	args := contracts_phase1.FindArgs{
		Path:     dir,
		Name:     &nameGlob,
		Type:     "file",
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}

	data, ok := env.Data.(contracts_phase1.FindData)
	if !ok {
		t.Fatalf("expected FindData, got %T", env.Data)
	}

	if data.Count != 2 {
		t.Errorf("expected 2 matches, got %d", data.Count)
	}
}

// INV-FIND-02: Policy Check
func TestFindTool_PolicyDenied(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits:  contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
		evalErr: &contracts.PolicyError{Reason: "denied"},
	}, &mockEnvelopeBuilder{})

	args := contracts_phase1.FindArgs{
		Path:     dir,
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure, got success")
	}

	if env.Error.Code != contracts.ErrPolicyDenied {
		t.Errorf("expected POLICY_DENIED, got %v", env.Error.Code)
	}
}

// INV-FIND-03: Missing Dir
func TestFindTool_MissingDir(t *testing.T) {
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
	}, &mockEnvelopeBuilder{})

	args := contracts_phase1.FindArgs{
		Path:     "/does/not/exist/ever",
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure, got success")
	}

	if env.Error.Code != contracts.ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %v", env.Error.Code)
	}
}

// INV-FIND-04: No Matches
func TestFindTool_NoMatches(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
	}, &mockEnvelopeBuilder{})

	nameGlob := "*.nonexistent"
	args := contracts_phase1.FindArgs{
		Path:     dir,
		Name:     &nameGlob,
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.FindData)
	if data.Count != 0 {
		t.Errorf("expected 0 matches, got %d", data.Count)
	}
	if len(data.Matches) != 0 {
		t.Errorf("expected empty array, got %v", data.Matches)
	}
}

// INV-FIND-05: Truncation
func TestFindTool_Truncation(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 2}, // Limit to 2 matches
	}, &mockEnvelopeBuilder{})

	args := contracts_phase1.FindArgs{
		Path:     dir,
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.FindData)
	if data.Count != 2 {
		t.Errorf("expected exactly 2 matches, got %d", data.Count)
	}
	if !env.Meta.Truncated {
		t.Errorf("expected Meta.Truncated to be true")
	}
}

// INV-FIND-06: Glob Error
func TestFindTool_GlobError(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
	}, &mockEnvelopeBuilder{})

	nameGlob := "[invalid"
	args := contracts_phase1.FindArgs{
		Path:     dir,
		Name:     &nameGlob,
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure, got success")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected INVALID_ARGS, got %v", env.Error.Code)
	}
}

// INV-FIND-07: No Symlink Follow
func TestFindTool_NoSymlinkFollow(t *testing.T) {
	dir := setupTempDir(t)
	tool := NewFindTool(&mockPolicyEngine{
		limits: contracts.PolicyLimits{TimeoutMs: 5000, MaxMatches: 100},
	}, &mockEnvelopeBuilder{})

	// Target the symlink directly to see if we walk its children
	args := contracts_phase1.FindArgs{
		Path:     dir,
		MaxDepth: 10,
	}

	env := tool.(interface {
		Execute(context.Context, contracts_phase1.FindArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected success, got error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.FindData)

	hasSymlinkDir := false
	hasChildOfSymlink := false

	for _, m := range data.Matches {
		if filepath.Base(m.Path) == "symlink_dir" {
			hasSymlinkDir = true
		}
		// "file4.txt" should only appear once (in sub/sub2), not inside symlink_dir
		if strings.Contains(m.Path, "symlink_dir") && filepath.Base(m.Path) == "file4.txt" {
			hasChildOfSymlink = true
		}
	}

	if !hasSymlinkDir {
		t.Errorf("symlink_dir should be reported")
	}
	if hasChildOfSymlink {
		t.Errorf("symlink should NOT be followed during walk")
	}
}
