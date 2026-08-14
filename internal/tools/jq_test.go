package tools

import (
	"context"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
)

func setupJqTest(maxOutputBytes int64) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedTools:  []string{"jq"},
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
	return NewJqTool(engine, builder)
}

func TestJq_ValidQuery(t *testing.T) {
	tool := setupJqTest(1024)
	args := contracts_phase1.JqArgs{
		Input:   `{"users":[{"name":"Alice"},{"name":"Bob"}]}`,
		Filter:  ".users[].name",
		Compact: true,
	}

	jq := tool.(*jqTool)
	env := jq.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data, ok := env.Data.(contracts_phase1.JqData)
	if !ok {
		t.Fatalf("expected JqData, got %T", env.Data)
	}

	if data.OutputType != "string" {
		t.Errorf("expected output_type string, got %s", data.OutputType)
	}

	// jq multiple results are usually wrapped in an array by our code if more than one
	expected := `["Alice","Bob"]`
	if data.Result != expected {
		t.Errorf("expected %s, got %s", expected, data.Result)
	}
}

func TestJq_InvalidJSON(t *testing.T) {
	tool := setupJqTest(1024)
	args := contracts_phase1.JqArgs{
		Input:  `{"users": invalid}`,
		Filter: ".users",
	}

	jq := tool.(*jqTool)
	env := jq.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for invalid json")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestJq_InvalidFilter(t *testing.T) {
	tool := setupJqTest(1024)
	args := contracts_phase1.JqArgs{
		Input:  `{"users":[{"name":"Alice"}]}`,
		Filter: ".users[", // invalid syntax
	}

	jq := tool.(*jqTool)
	env := jq.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for invalid filter")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestJq_OversizedInput(t *testing.T) {
	tool := setupJqTest(10) // Small limit
	args := contracts_phase1.JqArgs{
		Input:  `{"users":[{"name":"Alice"}]}`,
		Filter: ".",
	}

	jq := tool.(*jqTool)
	env := jq.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for oversized input")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestJq_TruncatedOutput(t *testing.T) {
	// The limit applies to BOTH input and output. We need input to be smaller than the limit, but output to be larger.
    // Wait, the test input is length 32: `{"users":"aaaaaaaaaaaaaaaaaaaa"}`
    // If the limit is 10, the input itself fails the MaxOutputBytes check.
    // Let's set the limit to 50 so input passes, but generate a larger output.
	tool2 := setupJqTest(100)
	args := contracts_phase1.JqArgs{
		Input:   `"1234567890"`, // len 12
		Filter:  "[.,.,.,.,.,.,.,.,.,.,.,.]", // duplicates it 12 times -> length > 100
		Compact: true,
	}

	jq := tool2.(*jqTool)
	env := jq.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK despite truncation, got: %v", env.Error)
	}

	if !env.Meta.Truncated {
		t.Errorf("expected truncated to be true")
	}

	data := env.Data.(contracts_phase1.JqData)
	if len(data.Result) != 100 {
		t.Errorf("expected length 100, got %d", len(data.Result))
	}
}
