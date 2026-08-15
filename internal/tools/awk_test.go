package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func TestAwkTool(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("A 10\nB 20\nC 30\n"), 0644)
	assert.NoError(t, err)

	csvFile := filepath.Join(tempDir, "data.csv")
	err = os.WriteFile(csvFile, []byte("id,name\n1,alice\n2,bob\n"), 0644)
	assert.NoError(t, err)

	emptyFile := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	assert.NoError(t, err)

	p := policy.DefaultPolicy()
	p.PolicyConfig.AllowedRoot = tempDir
	p.PolicyConfig.AllowedTools = append(p.PolicyConfig.AllowedTools, "awk")
	logger, _ := audit.NewLogger("stderr", "")
	engine := policy.NewEngine(p, logger)
	builder := response.NewBuilder()
	tool := NewAwkTool(engine, builder).(phase2.AwkTool)

	t.Run("Normal execution (column extraction)", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $2}",
			Path:    file1,
		}
		env := tool.Awk(args)
		assert.True(t, env.OK)
		data := env.Data.(phase2.AwkData)
		assert.Equal(t, "10\n20\n30\n", data.Output)
		assert.Equal(t, 3, data.LinesProcessed)
	})

	t.Run("Custom field separator", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program:        "{print $2}",
			Path:           csvFile,
			FieldSeparator: ",",
		}
		env := tool.Awk(args)
		assert.True(t, env.OK)
		data := env.Data.(phase2.AwkData)
		assert.Equal(t, "name\nalice\nbob\n", data.Output)
		assert.Equal(t, 3, data.LinesProcessed)
	})

	t.Run("Syntax error", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $2", // Missing closing brace
			Path:    file1,
		}
		env := tool.Awk(args)
		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
		assert.Contains(t, env.Error.Message, "syntax error")
	})

	t.Run("Write redirect blocked", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $1 > \"/etc/passwd\"}",
			Path:    file1,
		}
		env := tool.Awk(args)
		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
		assert.Contains(t, env.Error.Message, "AWK write redirects are not permitted")
	})

	t.Run("Path outside allowed root", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $1}",
			Path:    "/etc/passwd",
		}
		env := tool.Awk(args)
		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)
	})

	t.Run("Empty file", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $1}",
			Path:    emptyFile,
		}
		env := tool.Awk(args)
		assert.True(t, env.OK)
		data := env.Data.(phase2.AwkData)
		assert.Equal(t, "", data.Output)
		assert.Equal(t, 0, data.LinesProcessed)
	})

	t.Run("Output truncation", func(t *testing.T) {
		p.Limits.MaxOutputBytes = 5
		engine := policy.NewEngine(p, logger)
		tool := NewAwkTool(engine, builder).(phase2.AwkTool)

		args := phase2.AwkArgs{
			Program: "{print $0}",
			Path:    file1,
		}
		env := tool.Awk(args)
		assert.True(t, env.OK)
		data := env.Data.(phase2.AwkData)
		assert.Equal(t, "A 10\n", data.Output) // "A 10\n" is 5 bytes
		assert.True(t, env.Meta.Truncated)

		// Reset limit
		p.Limits.MaxOutputBytes = 524288
	})

	t.Run("Missing path", func(t *testing.T) {
		args := phase2.AwkArgs{
			Program: "{print $1}",
		}
		env := tool.Awk(args)
		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
	})

	t.Run("Missing program", func(t *testing.T) {
		args := phase2.AwkArgs{
			Path: file1,
		}
		env := tool.Awk(args)
		assert.False(t, env.OK)
		assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
	})

	t.Run("IsMutating returns false", func(t *testing.T) {
		assert.False(t, tool.(contracts.Tool).IsMutating())
	})

	t.Run("Name returns awk", func(t *testing.T) {
		assert.Equal(t, "awk", tool.(contracts.Tool).Name())
	})
}
