package tools

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase4 "github.com/osmcp/osmcp/docs/contracts/phase-4"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
)

func TestHashFileTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	err := os.WriteFile(testFile, content, 0644)
	assert.NoError(t, err)

	sha256Hash := sha256.Sum256(content)
	sha256Expected := hex.EncodeToString(sha256Hash[:])

	md5Hash := md5.Sum(content)
	md5Expected := hex.EncodeToString(md5Hash[:])

	sha1Hash := sha1.Sum(content)
	sha1Expected := hex.EncodeToString(sha1Hash[:])

	tests := []struct {
		name          string
		args          contracts_phase4.HashFileArgs
		policyDeny    bool
		policyTimeout int64
		expectedOK    bool
		expectedError contracts.ErrorCode
		expectedHash  string
		expectedAlg   string
	}{
		{
			name: "valid sha256 hash",
			args: contracts_phase4.HashFileArgs{
				Path:      testFile,
				Algorithm: "sha256",
			},
			expectedOK:   true,
			expectedHash: sha256Expected,
			expectedAlg:  "sha256",
		},
		{
			name: "valid md5 hash",
			args: contracts_phase4.HashFileArgs{
				Path:      testFile,
				Algorithm: "md5",
			},
			expectedOK:   true,
			expectedHash: md5Expected,
			expectedAlg:  "md5",
		},
		{
			name: "valid sha1 hash",
			args: contracts_phase4.HashFileArgs{
				Path:      testFile,
				Algorithm: "sha1",
			},
			expectedOK:   true,
			expectedHash: sha1Expected,
			expectedAlg:  "sha1",
		},
		{
			name: "default algorithm is sha256",
			args: contracts_phase4.HashFileArgs{
				Path: testFile,
			},
			expectedOK:   true,
			expectedHash: sha256Expected,
			expectedAlg:  "sha256",
		},
		{
			name: "empty path",
			args: contracts_phase4.HashFileArgs{
				Path: "",
			},
			expectedOK:    false,
			expectedError: contracts.ErrInvalidArgs,
		},
		{
			name: "unsupported algorithm",
			args: contracts_phase4.HashFileArgs{
				Path:      testFile,
				Algorithm: "unsupported",
			},
			expectedOK:    false,
			expectedError: contracts.ErrInvalidArgs,
		},
		{
			name: "policy denied",
			args: contracts_phase4.HashFileArgs{
				Path: filepath.Join("/etc", "passwd"),
			},
			policyDeny:    true,
			expectedOK:    false,
			expectedError: contracts.ErrPolicyDenied,
		},
		{
			name: "file not found",
			args: contracts_phase4.HashFileArgs{
				Path: filepath.Join(tmpDir, "missing.txt"),
			},
			expectedOK:    false,
			expectedError: contracts.ErrNotFound,
		},
		{
			name: "path is directory",
			args: contracts_phase4.HashFileArgs{
				Path: tmpDir,
			},
			expectedOK:    false,
			expectedError: contracts.ErrInvalidArgs,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := policy.DefaultPolicy()
			p.PolicyConfig.AllowedRoot = tmpDir
			p.PolicyConfig.AllowedTools = append(p.PolicyConfig.AllowedTools, "hash_file")
			if tc.policyTimeout > 0 {
				p.Limits.TimeoutMs = tc.policyTimeout
			}

			logger, _ := audit.NewLogger("stderr", "")
			engine := policy.NewEngine(p, logger)

			builder := response.NewBuilder()
			tool := NewHashFileTool(engine, builder)

			env := tool.(interface {
				Execute(ctx context.Context, args contracts_phase4.HashFileArgs) contracts.Envelope
			}).Execute(context.Background(), tc.args)

			assert.Equal(t, tc.expectedOK, env.OK)

			if tc.expectedOK {
				assert.Nil(t, env.Error)
				data, ok := env.Data.(contracts_phase4.HashFileData)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedHash, data.Hash)
				assert.Equal(t, tc.expectedAlg, data.Algorithm)
			} else {
				assert.NotNil(t, env.Error)
				assert.Equal(t, tc.expectedError, env.Error.Code)
			}
		})
	}
}

func TestHashFileTool_Timeout(t *testing.T) {
	// Let's create a large file to hash
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	f, err := os.Create(testFile)
	assert.NoError(t, err)
	// Write 1MB of zeroes
	chunk := make([]byte, 1024*1024)
	_, err = f.Write(chunk)
	assert.NoError(t, err)
	f.Close()

	p := policy.DefaultPolicy()
	p.PolicyConfig.AllowedRoot = tmpDir
	p.PolicyConfig.AllowedTools = append(p.PolicyConfig.AllowedTools, "hash_file")
	// Give a very small timeout
	p.Limits.TimeoutMs = 1
	logger, _ := audit.NewLogger("stderr", "")
	engine := policy.NewEngine(p, logger)

	builder := response.NewBuilder()
	tool := NewHashFileTool(engine, builder)

	// Also use an already-cancelled context to ensure instant timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := tool.(interface {
		Execute(ctx context.Context, args contracts_phase4.HashFileArgs) contracts.Envelope
	}).Execute(ctx, contracts_phase4.HashFileArgs{
		Path: testFile,
	})

	assert.False(t, env.OK)
	assert.NotNil(t, env.Error)
	assert.Equal(t, contracts.ErrTimeout, env.Error.Code)
}
