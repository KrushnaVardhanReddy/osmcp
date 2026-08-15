package tools

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"strings"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/docs/contracts/phase-2"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

type mockPolicyEngineTar struct {
	allowedRoot    string
	maxOutputBytes int64
	maxMatches     int
}

func (m *mockPolicyEngineTar) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	for _, p := range pathArgs {
		if !strings.HasPrefix(p, m.allowedRoot) {
			return &contracts.PolicyError{Reason: "path outside allowed root"}
		}
	}
	return nil
}

func (m *mockPolicyEngineTar) IsToolVisible(toolName string) bool {
	return true
}

func (m *mockPolicyEngineTar) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{
		MaxOutputBytes: m.maxOutputBytes,
		MaxMatches:     m.maxMatches,
	}
}

func setupTarTest() (contracts.Tool, string) {
	tempDir, _ := os.MkdirTemp("", "osmcp-tar-test-*")
	pol := &mockPolicyEngineTar{
		allowedRoot: tempDir,
		maxMatches:  10,
		maxOutputBytes: 1024,
	}
	builder := response.NewBuilder()
	return NewTarTool(pol, builder), tempDir
}

func createArchive(t *testing.T, path string, files map[string][]byte, comp string) {
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	var tw *tar.Writer
	if comp == "gz" {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else if comp == "xz" {
		xw, err := xz.NewWriter(f)
		require.NoError(t, err)
		defer xw.Close()
		tw = tar.NewWriter(xw)
	} else {
		tw = tar.NewWriter(f)
	}
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write(content)
		require.NoError(t, err)
	}
}

func TestTarTool_List(t *testing.T) {
	tool, tempDir := setupTarTest()
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "test.tar.gz")
	createArchive(t, archivePath, map[string][]byte{
		"file1.txt": []byte("hello"),
		"dir1/":     []byte(""),
	}, "gz")

	req := phase2.TarArgs{
		Path:   archivePath,
		Action: phase2.TarActionList,
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.True(t, res.OK)

	data := res.Data.(phase2.TarListData)

	assert.Equal(t, 2, data.Count)
	assert.Contains(t, []string{"file1.txt", "dir1/"}, data.Entries[0].Name)
	assert.Contains(t, []string{"file1.txt", "dir1/"}, data.Entries[1].Name)
	// verify one is dir and one is not
	assert.True(t, data.Entries[0].IsDir != data.Entries[1].IsDir)
}

func TestTarTool_ExtractText(t *testing.T) {
	tool, tempDir := setupTarTest()
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "test.tar")
	createArchive(t, archivePath, map[string][]byte{
		"file1.txt": []byte("hello world"),
	}, "")

	req := phase2.TarArgs{
		Path:   archivePath,
		Action: phase2.TarActionExtract,
		Entry:  "file1.txt",
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.True(t, res.OK)

	data := res.Data.(phase2.TarExtractData)

	assert.Equal(t, "file1.txt", data.Entry)
	assert.Equal(t, "hello world", data.Content)
}

func TestTarTool_ExtractBinary(t *testing.T) {
	tool, tempDir := setupTarTest()
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "test.tar.xz")
	createArchive(t, archivePath, map[string][]byte{
		"binfile": {0x00, 0x01, 0xFF, 0xFE}, // invalid utf8
	}, "xz")

	req := phase2.TarArgs{
		Path:   archivePath,
		Action: phase2.TarActionExtract,
		Entry:  "binfile",
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.False(t, res.OK)
	assert.Equal(t, contracts.ErrExecFailed, res.Error.Code)
	assert.Contains(t, res.Error.Message, "binary file")
}

func TestTarTool_PathTraversal(t *testing.T) {
	tool, tempDir := setupTarTest()
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "evil.tar")
	createArchive(t, archivePath, map[string][]byte{
		"../../etc/passwd": []byte("root"),
	}, "")

	req := phase2.TarArgs{
		Path:   archivePath,
		Action: phase2.TarActionList,
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.False(t, res.OK)
	assert.Equal(t, contracts.ErrExecFailed, res.Error.Code)
	assert.Contains(t, res.Error.Message, "path traversal")
}

func TestTarTool_PolicyDenied(t *testing.T) {
	tool, _ := setupTarTest()

	req := phase2.TarArgs{
		Path:   "/etc/passwd", // outside allowed root
		Action: phase2.TarActionList,
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.False(t, res.OK)
	assert.Equal(t, contracts.ErrPolicyDenied, res.Error.Code)
}

func TestTarTool_NotFound(t *testing.T) {
	tool, tempDir := setupTarTest()
	defer os.RemoveAll(tempDir)

	req := phase2.TarArgs{
		Path:   filepath.Join(tempDir, "missing.tar"),
		Action: phase2.TarActionList,
	}

	tarTool := tool.(*tarTool)
	res := tarTool.Tar(req)
	assert.False(t, res.OK)
	assert.Equal(t, contracts.ErrNotFound, res.Error.Code)
}

func toMap(req phase2.TarArgs) map[string]interface{} {
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

func (m *mockPolicyEngineTar) RunScriptConfig() contracts.RunScriptConfig {
	return contracts.RunScriptConfig{}
}

func (m *mockPolicyEngineTar) AllowedRoot() string {
	return "/tmp/mockroot"
}
