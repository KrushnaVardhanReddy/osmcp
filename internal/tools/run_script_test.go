package tools

import (
	"context"
	"testing"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase3 "github.com/osmcp/osmcp/docs/contracts/phase-3"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
)

func TestRunScript(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name          string
		allowRun      bool
		script        string
		workingDir    string
		timeout       int
		expectOK      bool
		expectErrCode contracts.ErrorCode
		validateResp  func(t *testing.T, data contracts_phase3.RunScriptData, meta contracts.Meta)
	}{
		{
			name:          "allow_run_script = false returns POLICY_DENIED",
			allowRun:      false,
			script:        "echo hello",
			workingDir:    tempDir,
			expectOK:      false,
			expectErrCode: contracts.ErrPolicyDenied,
		},
		{
			name:          "blocked binary returns POLICY_DENIED",
			allowRun:      true,
			script:        "curl http://example.com",
			workingDir:    tempDir,
			expectOK:      false,
			expectErrCode: contracts.ErrPolicyDenied,
		},
		{
			name:          "successful script execution",
			allowRun:      true,
			script:        "echo hello stdout && echo hello stderr >&2",
			workingDir:    tempDir,
			expectOK:      true,
			validateResp: func(t *testing.T, data contracts_phase3.RunScriptData, meta contracts.Meta) {
				if data.Stdout != "hello stdout\n" {
					t.Errorf("expected stdout 'hello stdout\n', got %q", data.Stdout)
				}
				if data.Stderr != "hello stderr\n" {
					t.Errorf("expected stderr 'hello stderr\n', got %q", data.Stderr)
				}
				if data.ExitCode != 0 {
					t.Errorf("expected exit code 0, got %d", data.ExitCode)
				}
			},
		},
		{
			name:          "timeout fires and timed_out = true",
			allowRun:      true,
			script:        "sleep 5",
			workingDir:    tempDir,
			timeout:       100, // 100 ms
			expectOK:      true, // execution envelope is OK, script timed out
			validateResp: func(t *testing.T, data contracts_phase3.RunScriptData, meta contracts.Meta) {
				if !data.TimedOut {
					t.Errorf("expected timed_out = true")
				}
				if data.ExitCode == 0 {
					t.Errorf("expected non-zero exit code due to kill")
				}
			},
		},
		{
			name:          "output is truncated when it exceeds max_output_bytes",
			allowRun:      true,
			script:        "echo 1234567890", // 11 bytes with newline
			workingDir:    tempDir,
			expectOK:      true,
			validateResp: func(t *testing.T, data contracts_phase3.RunScriptData, meta contracts.Meta) {
				if !meta.Truncated {
					t.Errorf("expected truncated = true")
				}
				if len(data.Stdout) > 5 {
					t.Errorf("expected stdout to be truncated to 5 bytes, got %d bytes", len(data.Stdout))
				}
			},
		},
		{
			name:          "working_dir outside allowed_root returns POLICY_DENIED",
			allowRun:      true,
			script:        "echo hello",
			workingDir:    "/tmp", // outside tempDir
			expectOK:      false,
			expectErrCode: contracts.ErrPolicyDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := policy.DefaultPolicy()
			p.PolicyConfig.AllowedRoot = tempDir
			p.PolicyConfig.AllowRunScript = tc.allowRun
			p.PolicyConfig.AllowMutation = true
			p.RunScript.AllowNetwork = true
			p.PolicyConfig.AllowedTools = append(p.PolicyConfig.AllowedTools, "run_script")
			if tc.name == "output is truncated when it exceeds max_output_bytes" {
				p.Limits.MaxOutputBytes = 5
			} else {
				p.Limits.MaxOutputBytes = 524288
			}

			engine := policy.NewEngine(p, nil)
			builder := response.NewBuilder()
			tool := NewRunScriptTool(engine, builder)

			args := contracts_phase3.RunScriptArgs{
				Interpreter: "bash",
				Script:      tc.script,
				WorkingDir:  tc.workingDir,
				TimeoutMs:   tc.timeout,
			}

			// We need to call execute directly as it returns the envelope.
			// Or we can cast and call.
			typedTool, ok := tool.(*runScriptTool)
			if !ok {
				t.Fatalf("could not cast to runScriptTool")
			}

			ctx := context.Background()
			env := typedTool.execute(ctx, args)

			if env.OK != tc.expectOK {
				t.Fatalf("expected OK=%v, got OK=%v (error: %v)", tc.expectOK, env.OK, env.Error)
			}

			if !tc.expectOK && tc.expectErrCode != "" {
				if env.Error == nil {
					t.Fatalf("expected error code %s but error was nil", tc.expectErrCode)
				}
				if env.Error.Code != tc.expectErrCode {
					t.Errorf("expected error code %s, got %s", tc.expectErrCode, env.Error.Code)
				}
			}

			if tc.expectOK && tc.validateResp != nil {
				data, ok := env.Data.(contracts_phase3.RunScriptData)
				if !ok {
					t.Fatalf("expected RunScriptData payload, got %T", env.Data)
				}
				tc.validateResp(t, data, env.Meta)
			}
		})
	}
}
