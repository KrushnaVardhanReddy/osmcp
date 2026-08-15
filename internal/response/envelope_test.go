package response_test

import (
	"encoding/json"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/stretchr/testify/assert"
)

func TestEnvelopeBuilder_Success(t *testing.T) {
	builder := response.NewBuilder()

	tool := "grep"
	data := map[string]interface{}{
		"matches": []interface{}{
			map[string]interface{}{
				"file": "src/main.go",
				"line": 42.0, // json decodes numbers to float64
				"text": "// TODO: fix this",
			},
		},
		"count": 1.0,
	}
	meta := contracts.Meta{
		ExecutionTimeMs: 12,
		Truncated:       false,
	}

	env := builder.Success(tool, data, meta)

	jsonBytes, err := json.Marshal(env)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "1", parsed["version"])
	assert.Equal(t, true, parsed["ok"])
	assert.Equal(t, "grep", parsed["tool"])
	assert.Equal(t, data, parsed["data"])
	assert.Nil(t, parsed["error"])

	metaParsed := parsed["meta"].(map[string]interface{})
	assert.Equal(t, 12.0, metaParsed["execution_time_ms"])
	assert.Equal(t, false, metaParsed["truncated"])
}

func TestEnvelopeBuilder_Failure(t *testing.T) {
	builder := response.NewBuilder()

	tool := "rm"
	meta := contracts.Meta{
		ExecutionTimeMs: 2,
		Truncated:       false,
	}

	env := builder.Failure(tool, contracts.ErrPolicyDenied, "Path '/etc/passwd' is outside the allowed root '/home/user/project'.", false, meta)

	jsonBytes, err := json.Marshal(env)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "1", parsed["version"])
	assert.Equal(t, false, parsed["ok"])
	assert.Equal(t, "rm", parsed["tool"])
	assert.Nil(t, parsed["data"])

	metaParsed := parsed["meta"].(map[string]interface{})
	assert.Equal(t, 2.0, metaParsed["execution_time_ms"])
	assert.Equal(t, false, metaParsed["truncated"])

	errorParsed := parsed["error"].(map[string]interface{})
	assert.Equal(t, "POLICY_DENIED", errorParsed["code"])
	assert.Equal(t, "Path '/etc/passwd' is outside the allowed root '/home/user/project'.", errorParsed["message"])
	assert.Equal(t, false, errorParsed["retryable"])
}
