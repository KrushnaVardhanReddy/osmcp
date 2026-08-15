package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
)

type logger struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewLogger creates a new AuditLogger.
// destination can be "stderr" or "file".
// if "file", path must be provided.
func NewLogger(destination, path string) (contracts.AuditLogger, error) {
	var w io.Writer

	if destination == "file" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}
		w = f
	} else {
		w = os.Stderr
	}

	return &logger{
		writer: w,
	}, nil
}

// NewLoggerWithWriter creates an AuditLogger with a custom writer (useful for tests).
func NewLoggerWithWriter(w io.Writer) contracts.AuditLogger {
	return &logger{
		writer: w,
	}
}

// Log writes a single audit entry as NDJSON.
func (l *logger) Log(entry contracts.AuditEntry) {
	type customAuditEntry struct {
		Timestamp      string   `json:"ts"`
		CallID         string   `json:"call_id"`
		Tool           string   `json:"tool"`
		PathArgs       []string `json:"path_args"`
		PolicyDecision string   `json:"policy_decision"`
		DenialCode     string   `json:"denial_code,omitempty"`
		DurationMs     int64    `json:"duration_ms"`
		OK             bool     `json:"ok"`
	}

	// RFC3339 with millisecond precision
	ts := entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
	if entry.PathArgs == nil {
		entry.PathArgs = []string{}
	}

	ce := customAuditEntry{
		Timestamp:      ts,
		CallID:         entry.CallID,
		Tool:           entry.Tool,
		PathArgs:       entry.PathArgs,
		PolicyDecision: entry.PolicyDecision,
		DenialCode:     entry.DenialCode,
		DurationMs:     entry.DurationMs,
		OK:             entry.OK,
	}

	bytes, err := json.Marshal(ce)
	if err != nil {
		return // ignore serialization failures
	}
	bytes = append(bytes, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.writer.Write(bytes)
}
