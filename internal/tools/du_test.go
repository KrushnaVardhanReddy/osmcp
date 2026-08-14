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

func setupDuTest(t *testing.T, allowedRoot string, allowedTools []string) contracts.Tool {
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
	return NewDuTool(engine, builder)
}

func TestDuTool_INV_UTIL_09_HumanFormat(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupDuTest(t, tempDir, []string{"du"})

	file := filepath.Join(tempDir, "file.txt")
	// Write 512 KB
	buf := make([]byte, 512*1024)
	os.WriteFile(file, buf, 0644)

	args := contracts_phase1.DuArgs{
		Path:     file,
		MaxDepth: 1,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.DuArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.DuData)
	assert.Equal(t, int64(512*1024), data.TotalBytes)
	assert.Equal(t, "512 KB", data.TotalHuman)
}

func TestDuTool_Success_Dir(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupDuTest(t, tempDir, []string{"du"})

	// Setup structure
	os.MkdirAll(filepath.Join(tempDir, "internal", "policy"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "docs"), 0755)

	// internal/file1.txt: 100 KB
	os.WriteFile(filepath.Join(tempDir, "internal", "file1.txt"), make([]byte, 100*1024), 0644)

	// internal/policy/file2.txt: 100 KB
	os.WriteFile(filepath.Join(tempDir, "internal", "policy", "file2.txt"), make([]byte, 100*1024), 0644)

	// docs/file3.txt: 312 KB
	os.WriteFile(filepath.Join(tempDir, "docs", "file3.txt"), make([]byte, 312*1024), 0644)

	args := contracts_phase1.DuArgs{
		Path:     tempDir,
		MaxDepth: 1,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.DuArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.DuData)

	assert.Equal(t, int64(512*1024), data.TotalBytes)
	assert.Equal(t, "512 KB", data.TotalHuman)

	assert.Equal(t, 2, len(data.Breakdown)) // internal/ and docs/
	for _, entry := range data.Breakdown {
		if entry.Path == "internal/" {
			assert.Equal(t, int64(200*1024), entry.Bytes)
			assert.Equal(t, "200 KB", entry.Human)
		} else if entry.Path == "docs/" {
			assert.Equal(t, int64(312*1024), entry.Bytes)
			assert.Equal(t, "312 KB", entry.Human)
		} else {
			t.Fatalf("unexpected breakdown path: %s", entry.Path)
		}
	}
}

func TestDuTool_PolicyDenied(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupDuTest(t, "/tmp/allowed", []string{"du"})

	args := contracts_phase1.DuArgs{
		Path:     tempDir,
		MaxDepth: 1,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.DuArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}
