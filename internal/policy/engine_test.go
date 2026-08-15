package policy_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuditLogger implementation for testing
type mockAuditLogger struct {
	entries []contracts.AuditEntry
}

func (m *mockAuditLogger) Log(entry contracts.AuditEntry) {
	m.entries = append(m.entries, entry)
}

func setupTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "osmcp-policy-test-*")
	require.NoError(t, err)

	// Create some files and symlinks for testing
	err = os.MkdirAll(filepath.Join(dir, "myproject", "src"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "myproject", "src", "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)

	// Escape path (outside root)
	err = os.MkdirAll(filepath.Join(dir, "etc"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "etc", "passwd"), []byte("secret"), 0644)
	require.NoError(t, err)

	// Symlink pointing outside
	err = os.Symlink(filepath.Join(dir, "etc", "passwd"), filepath.Join(dir, "myproject", "passwd_link"))
	require.NoError(t, err)

	// Symlink dir pointing outside
	err = os.Symlink(filepath.Join(dir, "etc"), filepath.Join(dir, "myproject", "etc_link"))
	require.NoError(t, err)

	return dir
}

func TestPolicyEngine_Evaluate(t *testing.T) {
	testDir := setupTestDir(t)
	defer os.RemoveAll(testDir)

	allowedRoot := filepath.Join(testDir, "myproject")

	pol := policy.DefaultPolicy()
	pol.PolicyConfig.AllowedRoot = allowedRoot
	pol.PolicyConfig.AllowedTools = append(pol.PolicyConfig.AllowedTools, "rm", "git_add", "cp")
	pol.PolicyConfig.AllowMutation = false
	pol.PolicyConfig.AllowGitWrite = false

	audit := &mockAuditLogger{}
	engine := policy.NewEngine(pol, audit)

	ctx := context.Background()

	tests := []struct {
		name       string
		toolName   string
		pathArgs   []string
		isMutating bool
		wantErr    bool
		errCode    string
		errMsg     string
	}{
		// Case 1: Tool not in allowlist
		{"tool not in allowlist", "unknown_tool", []string{}, false, true, "POLICY_DENIED", "tool not in allowlist"},
		// Case 2: Allowed path (inside root)
		{"allowed path inside root", "grep", []string{filepath.Join(allowedRoot, "src", "main.go")}, false, false, "", ""},
		// Case 3: Path outside root
		{"path outside root", "grep", []string{filepath.Join(testDir, "etc", "passwd")}, false, true, "POLICY_DENIED", "path outside allowed root"},
		// Case 4: Path exactly allowed root
		{"path exactly allowed root", "ls", []string{allowedRoot}, false, false, "", ""},
		// Case 5: Path is root's prefix but not inside (e.g. /myproject-2 when /myproject is allowed)
		{"path prefix match but different dir", "ls", []string{allowedRoot + "-2"}, false, true, "POLICY_DENIED", "path outside allowed root"},
		// Case 6: Symlink pointing outside allowed root
		{"symlink outside root", "cat", []string{filepath.Join(allowedRoot, "passwd_link")}, false, true, "POLICY_DENIED", "path outside allowed root"},
		// Case 7: Non-existent path inside root (should still pass policy if prefix matches)
		{"non-existent path inside root", "cat", []string{filepath.Join(allowedRoot, "not_found.txt")}, false, false, "", ""},
		// Case 8: Relative path traversal attempting to escape (e.g. /home/user/myproject/../../etc/passwd)
		{"path traversal escape", "cat", []string{filepath.Join(allowedRoot, "..", "..", "etc", "passwd")}, false, true, "POLICY_DENIED", "path outside allowed root"},
		// Case 9: Mutation allowed/denied
		{"mutation denied", "rm", []string{filepath.Join(allowedRoot, "src", "main.go")}, true, true, "POLICY_DENIED", "mutation not permitted"},
		// Case 10: Git write allowed/denied
		{"git write denied", "git_add", []string{}, false, true, "POLICY_DENIED", "git write not permitted"},
		// Case 11: Symlink directory pointing outside where suffix path doesn't exist (Bypass check)
		{"symlink dir outside non-existent suffix", "cp", []string{filepath.Join(allowedRoot, "etc_link", "new_file.txt")}, false, true, "POLICY_DENIED", "path outside allowed root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Evaluate(ctx, tt.toolName, tt.pathArgs, tt.isMutating)
			if tt.wantErr {
				require.Error(t, err)
				require.IsType(t, &contracts.PolicyError{}, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}

	// Verify Audit Logs
	require.Equal(t, len(tests), len(audit.entries))
	for i, tt := range tests {
		assert.Equal(t, tt.toolName, audit.entries[i].Tool)
		if tt.wantErr {
			assert.Equal(t, "DENIED", audit.entries[i].PolicyDecision)
			assert.Equal(t, tt.errCode, audit.entries[i].DenialCode)
			assert.False(t, audit.entries[i].OK)
		} else {
			assert.Equal(t, "ALLOW", audit.entries[i].PolicyDecision)
			assert.Empty(t, audit.entries[i].DenialCode)
			assert.True(t, audit.entries[i].OK)
		}
	}
}

func TestPolicyEngine_Evaluate_AllowMutation(t *testing.T) {
	testDir := setupTestDir(t)
	defer os.RemoveAll(testDir)
	allowedRoot := filepath.Join(testDir, "myproject")

	pol := policy.DefaultPolicy()
	pol.PolicyConfig.AllowedRoot = allowedRoot
	pol.PolicyConfig.AllowedTools = append(pol.PolicyConfig.AllowedTools, "rm", "git_add")
	pol.PolicyConfig.AllowMutation = true
	pol.PolicyConfig.AllowGitWrite = true

	audit := &mockAuditLogger{}
	engine := policy.NewEngine(pol, audit)
	ctx := context.Background()

	// Case 12: Mutation allowed
	err := engine.Evaluate(ctx, "rm", []string{filepath.Join(allowedRoot, "src", "main.go")}, true)
	require.NoError(t, err)

	// Case 13: Git write allowed
	err = engine.Evaluate(ctx, "git_add", []string{}, false)
	require.NoError(t, err)
}

func TestPolicyValidation(t *testing.T) {
	tests := []struct {
		name       string
		policy     policy.Policy
		wantErr    bool
		errMessage []string
	}{
		{
			name: "Valid policy",
			policy: policy.Policy{
				PolicyConfig: policy.PolicySection{
					AllowedRoot:  "/tmp",
					AllowedTools: []string{"grep", "ls"},
				},
				Limits: policy.LimitsSection{
					TimeoutMs:      1000,
					MaxOutputBytes: 1024 * 1024,
				},
				Audit: policy.AuditSection{
					Destination: "stderr",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing allowed_root",
			policy: policy.Policy{
				PolicyConfig: policy.PolicySection{
					AllowedRoot:  "",
					AllowedTools: []string{"grep"},
				},
				Limits: policy.LimitsSection{
					TimeoutMs:      1000,
					MaxOutputBytes: 1024 * 1024,
				},
				Audit: policy.AuditSection{Destination: "stderr"},
			},
			wantErr: true,
			errMessage: []string{"allowed_root is required"},
		},
		{
			name: "Relative allowed_root",
			policy: policy.Policy{
				PolicyConfig: policy.PolicySection{
					AllowedRoot:  "tmp",
					AllowedTools: []string{"grep"},
				},
				Limits: policy.LimitsSection{TimeoutMs: 1000, MaxOutputBytes: 1024 * 1024},
				Audit: policy.AuditSection{Destination: "stderr"},
			},
			wantErr: true,
			errMessage: []string{"allowed_root must be an absolute path: tmp"},
		},
		{
			name: "Unknown tool",
			policy: policy.Policy{
				PolicyConfig: policy.PolicySection{
					AllowedRoot:  "/tmp",
					AllowedTools: []string{"grep", "unknown_tool"},
				},
				Limits: policy.LimitsSection{TimeoutMs: 1000, MaxOutputBytes: 1024 * 1024},
				Audit: policy.AuditSection{Destination: "stderr"},
			},
			wantErr: true,
			errMessage: []string{"unknown tool in allowed_tools: unknown_tool"},
		},
		{
			name: "Limits out of range",
			policy: policy.Policy{
				PolicyConfig: policy.PolicySection{
					AllowedRoot:  "/tmp",
					AllowedTools: []string{"grep"},
				},
				Limits: policy.LimitsSection{
					TimeoutMs:      10, // too low
					MaxOutputBytes: 10, // too low
				},
				Audit: policy.AuditSection{Destination: "stderr"},
			},
			wantErr: true,
			errMessage: []string{
				"timeout_ms must be between 100 and 60000, got: 10",
				"max_output_bytes must be between 1024 and 10485760, got: 10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := policy.Validate(&tt.policy)
			if tt.wantErr {
				require.NotEmpty(t, errs)
				for i, msg := range tt.errMessage {
					assert.Contains(t, errs[i].Error(), msg)
				}
			} else {
				require.Empty(t, errs)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	pol, err := policy.LoadFromFile("../../testdata/policies/readonly.toml")
	require.NoError(t, err)
	assert.Equal(t, "/home/user/myproject", pol.PolicyConfig.AllowedRoot)
	assert.False(t, pol.PolicyConfig.AllowMutation)
	assert.Equal(t, int64(10000), pol.Limits.TimeoutMs)
	assert.Equal(t, "stderr", pol.Audit.Destination)
}
