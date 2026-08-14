package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func setupGrepTest(t *testing.T, allowedRoot string, allowedTools []string) contracts.Tool {
	if allowedRoot == "" {
		cwd, _ := os.Getwd()
		allowedRoot = filepath.Join(cwd, "..", "..", "testdata", "fixtures")
	}

	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedRoot:  allowedRoot,
			AllowedTools: allowedTools,
		},
		Limits: policy.LimitsSection{
			TimeoutMs:      1000,
			MaxOutputBytes: 1024,
			MaxMatches:     10,
		},
		Audit: policy.AuditSection{
			Destination: "stderr",
		},
	}

	logger := audit.NewLoggerWithWriter(os.Stderr)
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	return NewGrepTool(engine, builder)
}

func getFixturesDir() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "..", "testdata", "fixtures")
}

func TestGrepTool_INV_01_NoMatches(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern:       "nonexistent_string_123",
		Path:          filepath.Join(getFixturesDir(), "sample.txt"),
		Recursive:     false,
		CaseSensitive: true,
		Literal:       true,
		ContextLines:  0,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data, ok := env.Data.(contracts_phase1.GrepData)
	assert.True(t, ok)
	assert.Empty(t, data.Matches) // MUST be empty array, not nil
	assert.NotNil(t, data.Matches)
	assert.Equal(t, 0, data.Count)
}

func TestGrepTool_INV_02_Truncated(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern:       "line",
		Path:          filepath.Join(getFixturesDir(), "sample.txt"),
		Recursive:     false,
		CaseSensitive: true,
		Literal:       true,
		ContextLines:  0,
		MaxMatches:    2,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)
	assert.Equal(t, 2, data.Count)
	assert.True(t, env.Meta.Truncated)
}

func TestGrepTool_INV_03_OutsideAllowedRoot(t *testing.T) {
	gt := setupGrepTest(t, "/tmp/allowed", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern: "TODO",
		Path:    filepath.Join(getFixturesDir(), "sample.go"),
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGrepTool_INV_04_NotInAllowedTools(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"find"}) // grep not allowed

	args := contracts_phase1.GrepArgs{
		Pattern: "TODO",
		Path:    filepath.Join(getFixturesDir(), "sample.go"),
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestGrepTool_INV_05_NonExistentPath(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern: "TODO",
		Path:    filepath.Join(getFixturesDir(), "does_not_exist.go"),
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrNotFound, env.Error.Code)
}

func TestGrepTool_INV_06_EmptyPattern(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern: "",
		Path:    filepath.Join(getFixturesDir(), "sample.txt"),
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestGrepTool_INV_07_Timeout(t *testing.T) {
	// Timeout is tested by enforcing a very strict timeout policy
	// and using a slow context, or by mocking it, but since no mocks:
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedRoot:  getFixturesDir(),
			AllowedTools: []string{"grep"},
		},
		Limits: policy.LimitsSection{
			TimeoutMs: 1, // 1 ms
		},
	}
	logger := audit.NewLoggerWithWriter(os.Stderr)
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	gt := NewGrepTool(engine, builder)

	args := contracts_phase1.GrepArgs{
		Pattern: "TODO",
		Path:    getFixturesDir(), // scanning dir might take > 1ms
	}

	// A fast system might still pass this within 1ms.
	// We can simulate timeout by passing a canceled context or just accepting if it fails or succeeds,
	// but the spec says "Timeout simulation -> TIMEOUT".
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(ctx, args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrTimeout, env.Error.Code)
}

func TestGrepTool_INV_08_ShellInjection(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern: "; rm -rf /tmp/test",
		Path:    filepath.Join(getFixturesDir(), "sample.go"),
		Literal: true,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	// Should not execute shell, just search and find no matches.
	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)
	assert.Empty(t, data.Matches)
}

func TestGrepTool_INV_09_ContextLines(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern:      "line 5 is the target.",
		Path:         filepath.Join(getFixturesDir(), "sample.txt"),
		ContextLines: 2,
		Literal:      true,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)
	assert.Equal(t, 1, data.Count)
	match := data.Matches[0]

	// Should have 2 lines before and 2 after
	assert.Equal(t, 2, len(match.ContextBefore))
	assert.Equal(t, "it also has multiple lines for context testing.", match.ContextBefore[0])
	assert.Equal(t, "line 4 is here.", match.ContextBefore[1])

	assert.Equal(t, 2, len(match.ContextAfter))
	assert.Equal(t, "line 6 comes after.", match.ContextAfter[0])
	assert.Equal(t, "line 7 is here.", match.ContextAfter[1])
}

func TestGrepTool_INV_10_Literal(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	// Literal=true prevents regex interpreting ".*"
	args := contracts_phase1.GrepArgs{
		Pattern: ".*",
		Path:    filepath.Join(getFixturesDir(), "sample.txt"),
		Literal: true,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)
	assert.Equal(t, 1, data.Count)
	assert.Contains(t, data.Matches[0].Text, ".*")
}

func TestGrepTool_Recursive(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern:   "TODO",
		Path:      getFixturesDir(),
		Recursive: true,
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)

	// Should find TODO in sample.go and nested/deep.go
	assert.Equal(t, 2, data.Count)
}

func TestGrepTool_Include(t *testing.T) {
	gt := setupGrepTest(t, "", []string{"grep"})

	args := contracts_phase1.GrepArgs{
		Pattern:   "TODO",
		Path:      getFixturesDir(),
		Recursive: true,
		Include:   "*.go", // Only go files
	}

	env := gt.(interface {
		Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.GrepData)
	assert.Equal(t, 2, data.Count)
}
