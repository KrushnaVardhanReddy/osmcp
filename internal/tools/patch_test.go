package tools_test

import (

	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/osmcp/osmcp/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchTool(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("hello\nworld\n"), 0644)
	require.NoError(t, err)

	pol := policy.DefaultPolicy()
	pol.PolicyConfig.AllowedRoot = tmpDir
	pol.PolicyConfig.AllowMutation = true

	auditLogger, _ := audit.NewLogger("stderr", "")
	engine := policy.NewEngine(pol, auditLogger)
	builder := response.NewBuilder()
	patchTool := tools.NewPatchTool(engine, builder).(contracts_phase2.PatchMutationTool)

	t.Run("successful patch", func(t *testing.T) {
		req := contracts_phase2.PatchRequest{
			Path: file1,
			Diff: `--- a/file1.txt
+++ b/file1.txt
@@ -1,2 +1,2 @@
 hello
-world
+friends
`,
		}

		env := patchTool.Patch(req)
		assert.True(t, env.OK)
		assert.Nil(t, env.Error)

		content, err := os.ReadFile(file1)
		require.NoError(t, err)
		assert.Equal(t, "hello\nfriends\n", string(content))
	})

	t.Run("mutation denied", func(t *testing.T) {
		pol.PolicyConfig.AllowMutation = false
		engineDeny := policy.NewEngine(pol, auditLogger)
		patchToolDeny := tools.NewPatchTool(engineDeny, builder).(contracts_phase2.PatchMutationTool)

		req := contracts_phase2.PatchRequest{
			Path: file1,
			Diff: `--- a/file1.txt
+++ b/file1.txt
@@ -1,2 +1,2 @@
 hello
-friends
+world
`,
		}

		env := patchToolDeny.Patch(req)
		assert.False(t, env.OK)
		assert.NotNil(t, env.Error)
		assert.Equal(t, contracts.ErrPolicyDenied, env.Error.Code)

		content, err := os.ReadFile(file1)
		require.NoError(t, err)
		assert.Equal(t, "hello\nfriends\n", string(content)) // Unchanged
	})

	t.Run("invalid diff", func(t *testing.T) {
		pol.PolicyConfig.AllowMutation = true // Reset to true
		req := contracts_phase2.PatchRequest{
			Path: file1,
			Diff: "this is not a valid diff format",
		}

		env := patchTool.Patch(req)
		assert.False(t, env.OK)
		assert.NotNil(t, env.Error)
		assert.Equal(t, contracts.ErrInvalidArgs, env.Error.Code)
	})

    t.Run("invalid path", func(t *testing.T) {
		req := contracts_phase2.PatchRequest{
			Path: filepath.Join(tmpDir, "nonexistent.txt"),
			Diff: `--- a/nonexistent.txt
+++ b/nonexistent.txt
@@ -1,2 +1,2 @@
 hello
-world
+friends
`,
		}

		env := patchTool.Patch(req)
		assert.False(t, env.OK)
		assert.NotNil(t, env.Error)
		assert.Equal(t, contracts.ErrExecFailed, env.Error.Code)
	})
}
