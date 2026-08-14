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

func setupSedTest(maxOutputBytes int64) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedTools:  []string{"sed"},
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
	return NewSedTool(engine, builder)
}

func TestSed_ValidReplaceGlobal(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "hello world hello",
		Expression: "s/hello/hi/g",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data, ok := env.Data.(contracts_phase1.SedData)
	if !ok {
		t.Fatalf("expected SedData, got %T", env.Data)
	}

	if data.ReplacementsMade != 2 {
		t.Errorf("expected 2 replacements, got %d", data.ReplacementsMade)
	}

	if data.Result != "hi world hi" {
		t.Errorf("expected 'hi world hi', got '%s'", data.Result)
	}
}

func TestSed_ValidReplaceFirst(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "hello world hello",
		Expression: "s/hello/hi/",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.SedData)
	if data.ReplacementsMade != 1 {
		t.Errorf("expected 1 replacement, got %d", data.ReplacementsMade)
	}

	if data.Result != "hi world hello" {
		t.Errorf("expected 'hi world hello', got '%s'", data.Result)
	}
}

func TestSed_CaseInsensitive(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "Hello world hello",
		Expression: "s/hello/hi/gi",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.SedData)
	if data.ReplacementsMade != 2 {
		t.Errorf("expected 2 replacements, got %d", data.ReplacementsMade)
	}

	if data.Result != "hi world hi" {
		t.Errorf("expected 'hi world hi', got '%s'", data.Result)
	}
}

func TestSed_CaptureGroups(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "hello world",
		Expression: "s/(hello) (world)/$2 $1/",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK, got Error: %v", env.Error)
	}

	data := env.Data.(contracts_phase1.SedData)
	if data.Result != "world hello" {
		t.Errorf("expected 'world hello', got '%s'", data.Result)
	}
}

func TestSed_InvalidExpression(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "hello world",
		Expression: "s/hello/hi", // missing trailing slash
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for invalid expression")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestSed_InvalidRegex(t *testing.T) {
	tool := setupSedTest(1024)
	args := contracts_phase1.SedArgs{
		Input:      "hello world",
		Expression: "s/([a-z]+/hi/", // unclosed group
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for invalid regex")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestSed_OversizedInput(t *testing.T) {
	tool := setupSedTest(10)
	args := contracts_phase1.SedArgs{
		Input:      "hello world hello world",
		Expression: "s/hello/hi/g",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if env.OK {
		t.Fatalf("expected failure for oversized input")
	}

	if env.Error.Code != contracts.ErrInvalidArgs {
		t.Errorf("expected %s, got %s", contracts.ErrInvalidArgs, env.Error.Code)
	}
}

func TestSed_TruncatedOutput(t *testing.T) {
	tool := setupSedTest(20)
	args := contracts_phase1.SedArgs{
		Input:      "abc",
		Expression: "s/abc/" + strings.Repeat("x", 25) + "/",
	}

	sed := tool.(*sedTool)
	env := sed.Execute(context.Background(), args)

	if !env.OK {
		t.Fatalf("expected OK despite truncation")
	}

	if !env.Meta.Truncated {
		t.Errorf("expected truncated to be true")
	}

	data := env.Data.(contracts_phase1.SedData)
	if len(data.Result) != 20 {
		t.Errorf("expected length 20, got %d", len(data.Result))
	}
}
