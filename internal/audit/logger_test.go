package audit_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_NDJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := audit.NewLoggerWithWriter(&buf)

	ts := time.Date(2026, 8, 14, 10, 30, 0, 0, time.UTC)
	entry := contracts.AuditEntry{
		Timestamp:      ts,
		CallID:         "c_xyz789",
		Tool:           "rm",
		PathArgs:       []string{"/etc/passwd"},
		PolicyDecision: "DENIED",
		DenialCode:     "POLICY_DENIED",
		DurationMs:     1,
		OK:             false,
	}

	logger.Log(entry)

	output := buf.String()
	assert.True(t, strings.HasSuffix(output, "\n"), "Output must be newline-terminated")

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "2026-08-14T10:30:00.000Z", parsed["ts"])
	assert.Equal(t, "c_xyz789", parsed["call_id"])
	assert.Equal(t, "rm", parsed["tool"])
	assert.Equal(t, "DENIED", parsed["policy_decision"])
	assert.Equal(t, "POLICY_DENIED", parsed["denial_code"])
	assert.Equal(t, 1.0, parsed["duration_ms"])
	assert.Equal(t, false, parsed["ok"])

	pathArgs := parsed["path_args"].([]interface{})
	require.Len(t, pathArgs, 1)
	assert.Equal(t, "/etc/passwd", pathArgs[0])
}

func TestLogger_Concurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := audit.NewLoggerWithWriter(&buf)

	var wg sync.WaitGroup
	numRoutines := 100
	numLogs := 10

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numLogs; j++ {
				logger.Log(contracts.AuditEntry{
					Timestamp:      time.Now(),
					CallID:         "test",
					Tool:           "ls",
					PathArgs:       []string{},
					PolicyDecision: "ALLOW",
					OK:             true,
				})
			}
		}(i)
	}

	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, numRoutines*numLogs)

	for _, line := range lines {
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(line), &parsed)
		require.NoError(t, err, "Each line must be valid JSON")
	}
}
