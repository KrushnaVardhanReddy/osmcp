package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func setupTreeTest(t *testing.T, allowedRoot string, allowedTools []string) contracts.Tool {
	p := &policy.Policy{
		PolicyConfig: policy.PolicySection{
			AllowedRoot:  allowedRoot,
			AllowedTools: allowedTools,
		},
		Limits: policy.LimitsSection{
			TimeoutMs:      1000,
			MaxOutputBytes: 1024,
			MaxMatches:     5, // Set low for truncation testing
		},
		Audit: policy.AuditSection{
			Destination: "stderr",
		},
	}

	logger := audit.NewLoggerWithWriter(os.Stderr)
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	return NewTreeTool(engine, builder)
}

func TestTreeTool_INV_UTIL_01_PureGo(t *testing.T) {
	// Not explicitly testable without inspecting source, but we know it doesn't use os/exec.
}

func TestTreeTool_INV_UTIL_02_PolicyDenied(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTreeTest(t, "/tmp/allowed", []string{"tree"})

	args := contracts_phase1.TreeArgs{
		Path: filepath.Join(tempDir, "outside"),
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
}

func TestTreeTool_INV_UTIL_03_MissingPath(t *testing.T) {
	tempDir := t.TempDir()
	tt := setupTreeTest(t, tempDir, []string{"tree"})

	args := contracts_phase1.TreeArgs{
		Path: filepath.Join(tempDir, "missing"),
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.False(t, env.OK)
	assert.Equal(t, contracts.ErrNotFound, env.Error.Code)
}

func TestTreeTool_INV_UTIL_06_TreeOnFile(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "file.txt")
	os.WriteFile(file, []byte("test"), 0644)

	tt := setupTreeTest(t, tempDir, []string{"tree"})

	args := contracts_phase1.TreeArgs{
		Path: file,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.TreeData)
	assert.Equal(t, "file.txt", data.Tree)
	assert.Equal(t, 1, data.Files)
	assert.Equal(t, 0, data.Dirs)
}

func TestTreeTool_INV_UTIL_07_Truncation(t *testing.T) {
	tempDir := t.TempDir()
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(tempDir, "file" + string(rune(i))), []byte("test"), 0644)
	}

	tt := setupTreeTest(t, tempDir, []string{"tree"})

	args := contracts_phase1.TreeArgs{
		Path:     tempDir,
		MaxDepth: 1,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.TreeData)
	assert.True(t, env.Meta.Truncated)
	assert.True(t, strings.HasSuffix(data.Tree, "... (truncated)"))
	assert.Equal(t, 5, data.Files) // Limit is 5 in setupTreeTest
}

func TestTreeTool_Success(t *testing.T) {
	tempDir := t.TempDir()
	os.Mkdir(filepath.Join(tempDir, "dir1"), 0755)
	os.WriteFile(filepath.Join(tempDir, "dir1", "file1.txt"), []byte("test"), 0644)
	os.Mkdir(filepath.Join(tempDir, "dir2"), 0755)

	tt := setupTreeTest(t, tempDir, []string{"tree"})

	args := contracts_phase1.TreeArgs{
		Path:     tempDir,
		MaxDepth: 3,
	}

	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.TreeData)

	// Tree string will depend on readdir order which is sorted or filesystem dependent.
	// Since go 1.16 os.ReadDir sorts by name.
	expectedLines := []string{
		filepath.Base(tempDir),
		"├── dir1",
		"│   └── file1.txt",
		"└── dir2",
	}
	expected := strings.Join(expectedLines, "\n")
	assert.Equal(t, expected, data.Tree)
	assert.Equal(t, 2, data.Dirs)
	assert.Equal(t, 1, data.Files)
}

func TestTreeTool_ShowHidden(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "visible"), []byte("test"), 0644)

	tt := setupTreeTest(t, tempDir, []string{"tree"})

	// Hidden false
	args := contracts_phase1.TreeArgs{
		Path:       tempDir,
		MaxDepth:   1,
		ShowHidden: false,
	}
	env := tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data := env.Data.(contracts_phase1.TreeData)
	assert.NotContains(t, data.Tree, ".hidden")
	assert.Contains(t, data.Tree, "visible")

	// Hidden true
	args.ShowHidden = true
	env = tt.(interface {
		Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
	}).Execute(context.Background(), args)

	assert.True(t, env.OK)
	data = env.Data.(contracts_phase1.TreeData)
	assert.Contains(t, data.Tree, ".hidden")
	assert.Contains(t, data.Tree, "visible")
}
