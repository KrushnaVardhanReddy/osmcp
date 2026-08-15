package tools

import (
	"context"
	"os"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/docs/contracts/phase-4"
	"github.com/osmcp/osmcp/internal/response"
)

type mockPolicyEngineGetEnvDenyAll struct{}

func (m *mockPolicyEngineGetEnvDenyAll) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	return &contracts.PolicyError{Reason: "policy denied"}
}
func (m *mockPolicyEngineGetEnvDenyAll) IsToolVisible(toolName string) bool { return true }
func (m *mockPolicyEngineGetEnvDenyAll) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{}
}
func (m *mockPolicyEngineGetEnvDenyAll) AllowedRoot() string { return "/tmp" }
func (m *mockPolicyEngineGetEnvDenyAll) RunScriptConfig() contracts.RunScriptConfig { return contracts.RunScriptConfig{} }

func TestGetEnvTool(t *testing.T) {
	builder := response.NewBuilder()

	// Setup environment variables for test
	os.Setenv("TEST_KEY_1", "VALUE1")
	os.Setenv("TEST_KEY_2", "VALUE2")
	os.Setenv("SECRET_KEY", "SECRET")

	t.Cleanup(func() {
		os.Unsetenv("TEST_KEY_1")
		os.Unsetenv("TEST_KEY_2")
		os.Unsetenv("SECRET_KEY")
	})

	tests := []struct {
		name          string
		allowlist     []string
		args          contracts_phase4.GetEnvArgs
		policyDeny    bool
		wantOK        bool
		wantErrorCode contracts.ErrorCode
		wantVars      map[string]string
	}{
		{
			name:       "Policy deny",
			policyDeny: true,
			wantOK:     false,
			wantErrorCode: contracts.ErrPolicyDenied,
		},
		{
			name:      "Empty allowlist should fail",
			allowlist: []string{},
			args:      contracts_phase4.GetEnvArgs{},
			wantOK:    false,
			wantErrorCode: contracts.ErrPolicyDenied,
		},
		{
			name:      "Return all explicitly allowed keys",
			allowlist: []string{"TEST_KEY_1", "TEST_KEY_2"},
			args:      contracts_phase4.GetEnvArgs{},
			wantOK:    true,
			wantVars:  map[string]string{"TEST_KEY_1": "VALUE1", "TEST_KEY_2": "VALUE2"},
		},
		{
			name:      "Return requested allowed keys",
			allowlist: []string{"TEST_KEY_1", "TEST_KEY_2"},
			args:      contracts_phase4.GetEnvArgs{Keys: []string{"TEST_KEY_1"}},
			wantOK:    true,
			wantVars:  map[string]string{"TEST_KEY_1": "VALUE1"},
		},
		{
			name:      "Do not return requested unallowed keys",
			allowlist: []string{"TEST_KEY_1"},
			args:      contracts_phase4.GetEnvArgs{Keys: []string{"TEST_KEY_1", "SECRET_KEY"}},
			wantOK:    true,
			wantVars:  map[string]string{"TEST_KEY_1": "VALUE1"},
		},
		{
			name:      "Request completely unallowed keys returns empty map",
			allowlist: []string{"TEST_KEY_1"},
			args:      contracts_phase4.GetEnvArgs{Keys: []string{"SECRET_KEY"}},
			wantOK:    true,
			wantVars:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var policy contracts.PolicyEngine
			if tt.policyDeny {
				policy = &mockPolicyEngineGetEnvDenyAll{}
			} else {
				policy = &mockPolicyEngine{allowedRoot: "/tmp"}
			}

			// We need to type assert the GetEnvTool since we want to call execute directly,
			// or we can test it through RegisterMCP but direct is cleaner for unit tests.
			toolRaw := NewGetEnvTool(policy, builder, tt.allowlist)
			tool := toolRaw.(*getEnvTool)

			env := tool.execute(context.Background(), tt.args)

			if env.OK != tt.wantOK {
				t.Fatalf("expected OK=%v, got %v", tt.wantOK, env.OK)
			}

			if !tt.wantOK {
				if env.Error == nil {
					t.Fatalf("expected error, got nil")
				}
				if env.Error.Code != tt.wantErrorCode {
					t.Errorf("expected error code %v, got %v", tt.wantErrorCode, env.Error.Code)
				}
				return
			}

			data, ok := env.Data.(contracts_phase4.GetEnvData)
			if !ok {
				t.Fatalf("expected GetEnvData, got %T", env.Data)
			}

			if len(data.Variables) != len(tt.wantVars) {
				t.Errorf("expected %d variables, got %d", len(tt.wantVars), len(data.Variables))
			}

			for k, v := range tt.wantVars {
				if data.Variables[k] != v {
					t.Errorf("expected %s=%s, got %s", k, v, data.Variables[k])
				}
			}
		})
	}
}
