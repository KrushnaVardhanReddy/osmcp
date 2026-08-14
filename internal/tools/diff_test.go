package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
)

func setupDiffTest(maxOutputBytes int64) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedTools:  []string{"diff"},
			AllowedRoot:   "/tmp",
			AllowMutation: false,
		},
		Limits: policy.LimitsSection{
			TimeoutMs:      1000,
			MaxOutputBytes: maxOutputBytes,
			MaxMatches:     100,
		},
	}
	engine := policy.NewEngine(p, nil)
	builder := response.NewBuilder()
	return NewDiffTool(engine, builder)
}

func TestDiff_ValidDifference(t *testing.T) {
	tool := setupDiffTest(1024)
	args := contracts_phase1.DiffArgs{
		A: "hello\nworld\nend",
		B: "hello\nGo\nend",
	}

	diff := tool.(*diffTool)
	env := diff.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data, ok := env.Data.(contracts_phase1.DiffData)
	if !ok {
		t.Fatalf("expected DiffData, got %T", env.Data)
	}

	if data.Identical {
		t.Errorf("expected identical to be false")
	}

	if data.Additions == 0 || data.Deletions == 0 {
		t.Errorf("expected additions and deletions > 0")
	}

	if len(data.Patch) == 0 {
		t.Errorf("expected non-empty patch")
	}
}

func TestDiff_IdenticalStrings(t *testing.T) {
	tool := setupDiffTest(1024)
	args := contracts_phase1.DiffArgs{
		A: "hello\nworld\nend",
		B: "hello\nworld\nend",
	}

	diff := tool.(*diffTool)
	env := diff.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.DiffData)
	if !data.Identical {
		t.Errorf("expected identical to be true")
	}

	if data.Additions != 0 || data.Deletions != 0 {
		t.Errorf("expected 0 additions and deletions")
	}

	if len(data.Patch) != 0 {
		t.Errorf("expected empty patch")
	}
}

func TestDiff_OversizedInput(t *testing.T) {
	tool := setupDiffTest(10)
	args := contracts_phase1.DiffArgs{
		A: "this string is longer than 10 bytes",
		B: "short",
	}

	diff := tool.(*diffTool)
	env := diff.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for oversized input")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestDiff_TruncatedOutput(t *testing.T) {
	tool := setupDiffTest(20)
	args := contracts_phase1.DiffArgs{
		A: "abc",
		B: "def" + strings.Repeat("g", 15),
	}

	diff := tool.(*diffTool)
	env := diff.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK despite truncation")
	}

	if !env.Meta.Truncated {
		t.Errorf("expected truncated to be true")
	}

	data := env.Data.(contracts_phase1.DiffData)
	if len(data.Patch) != 20 {
		t.Errorf("expected length 20, got %d", len(data.Patch))
	}
}
