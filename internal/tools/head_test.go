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

func setupHeadTest(t *testing.T, allowedRoot string, allowedTools []string) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedRoot:  allowedRoot,
			AllowedTools: allowedTools,
		},
		Limits: policy.LimitsSection{
			TimeoutMs:      1000,
			MaxOutputBytes: 10240,
			MaxMatches:     100,
		},
		Audit: policy.AuditSection{
			Destination: "stderr",
		},
	}

	logger := audit.NewLoggerWithWriter(os.Stderr)
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	return NewHeadTool(engine, builder)
}

func TestHeadTool_INV_UTIL_04_HeadOnDir(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupHeadTest(t, tempDir, []string{"head"})

	args := contracts_phase1.HeadArgs{
		Path:  tempDir,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestHeadTool_INV_UTIL_05_HeadOnBinary(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupHeadTest(t, tempDir, []string{"head"})

	binFile := filepath.Join(tempDir, "bin.dat")
	// write some null bytes to make it binary
	os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0x03}, 0644)

	args := contracts_phase1.HeadArgs{
		Path:  binFile,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
	assert.Contains(t, env.Error.Message, "Cannot read binary file")
}

func TestHeadTool_SuccessAndEOF(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupHeadTest(t, tempDir, []string{"head"})

	file := filepath.Join(tempDir, "file.txt")
	os.WriteFile(file, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	// EOF Reached
	args := contracts_phase1.HeadArgs{
		Path:  file,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.HeadData)
	assert.Equal(t, 5, data.LinesReturned)
	assert.True(t, data.EOFReached)
	assert.Equal(t, "line1\nline2\nline3\nline4\nline5", data.Content)

	// EOF Not Reached
	args.Lines = 3
	env = tt.(interface {
		Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data = env.Data.(contracts_phase1.HeadData)
	assert.Equal(t, 3, data.LinesReturned)
	assert.False(t, data.EOFReached)
	assert.Equal(t, "line1\nline2\nline3", data.Content)
}

func TestHeadTool_PolicyDenied(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupHeadTest(t, "/tmp/allowed", []string{"head"})

	args := contracts_phase1.HeadArgs{
		Path:  filepath.Join(tempDir, "file.txt"),
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}
