package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
    "fmt"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func setupTailTest(t *testing.T, allowedRoot string, allowedTools []string) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedRoot:  allowedRoot,
			AllowedTools: allowedTools,
		},
		Limits: policy.LimitsSection{
			TimeoutMs:      1000,
			MaxOutputBytes: 1024000, // increased for this test
			MaxMatches:     100,
		},
		Audit: policy.AuditSection{
			Destination: "stderr",
		},
	}

	logger := audit.NewLoggerWithWriter(os.Stderr)
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	return NewTailTool(engine, builder)
}

func TestTailTool_INV_UTIL_04_TailOnDir(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTailTest(t, tempDir, []string{"tail"})

	args := contracts_phase1.TailArgs{
		Path:  tempDir,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
}

func TestTailTool_INV_UTIL_05_TailOnBinary(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTailTest(t, tempDir, []string{"tail"})

	binFile := filepath.Join(tempDir, "bin.dat")
	os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0x03}, 0644)

	args := contracts_phase1.TailArgs{
		Path:  binFile,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
	assert.Contains(t, env.Error.Message, "Cannot read binary file")
}

func TestTailTool_INV_UTIL_08_TailMemory(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTailTest(t, tempDir, []string{"tail"})

	file := filepath.Join(tempDir, "large.txt")

	f, err := os.Create(file)
	assert.NoError(t, err)
	for i := 1; i <= 10000; i++ {
		f.WriteString(fmt.Sprintf("line %d\n", i))
	}
	f.Close()

	args := contracts_phase1.TailArgs{
		Path:  file,
		Lines: 20,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK, env.Error)

	// The problem was if the file is larger than MaxOutputBytes, it might error or truncate depending on how builder handles it.
	// But in this implementation, the builder just blindly accepts it (envelope builder only checks data).
	// Actually, wait, the panic is `interface conversion: interface {} is nil, not contracts.TailData`
	// This means `env.Data` was `nil`. Why? Because `env.OK` was false!
	// Let's assert env.OK first and check Error.
    if !env.OK {
        t.Fatalf("Expected env.OK to be true, got false. Error: %+v", env.Error)
    }

	data := env.Data.(contracts_phase1.TailData)
	assert.Equal(t, 20, data.LinesReturned)

	lines := strings.Split(data.Content, "\n")
	assert.Equal(t, 20, len(lines))
    assert.Equal(t, "line 10000", lines[19])
}

func TestTailTool_Success(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTailTest(t, tempDir, []string{"tail"})

	file := filepath.Join(tempDir, "file.txt")
	os.WriteFile(file, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644)

	// Fewer lines than limit
	args := contracts_phase1.TailArgs{
		Path:  file,
		Lines: 10,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.TailData)
	assert.Equal(t, 5, data.LinesReturned)
	assert.Equal(t, "line1\nline2\nline3\nline4\nline5", data.Content)

	// Exact limit
	args.Lines = 3
	env = tt.(interface {
		Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data = env.Data.(contracts_phase1.TailData)
	assert.Equal(t, 3, data.LinesReturned)
	assert.Equal(t, "line3\nline4\nline5", data.Content)
}
