package tools

import (
	"encoding/json"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase3 "github.com/osmcp/osmcp/docs/contracts/phase-3"
)

type runScriptTool struct {
	policy  contracts.PolicyEngine
	builder contracts.EnvelopeBuilder
}

// NewRunScriptTool creates a new run_script tool instance.
func NewRunScriptTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts.Tool {
	return &runScriptTool{
		policy:  policy,
		builder: builder,
	}
}

func (t *runScriptTool) Name() string {
	return "run_script"
}

func (t *runScriptTool) Description() string {
	return "Executes a shell script (bash or sh). This is a Tier 2 tool and must be explicitly permitted by policy."
}

func (t *runScriptTool) IsMutating() bool {
	return true
}

func (t *runScriptTool) RegisterMCP(s *server.MCPServer) {
	s.AddTool(mcp.NewTool(
		t.Name(),
		mcp.WithDescription(t.Description()),
		mcp.WithString("interpreter", mcp.Required(), mcp.Description("Must be one of 'bash' or 'sh'.")),
		mcp.WithString("script", mcp.Required(), mcp.Description("The shell script body to execute.")),
		mcp.WithString("working_dir", mcp.Description("Absolute path inside allowed_root to set as CWD. Defaults to allowed_root.")),
		mcp.WithNumber("timeout_ms", mcp.Description("Per-call timeout override (<= policy limit).")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid arguments type")
		}

		interpreter, _ := args["interpreter"].(string)
		script, _ := args["script"].(string)
		workingDir, _ := args["working_dir"].(string)

		timeoutMs := 0
		if val, exists := args["timeout_ms"]; exists {
			switch v := val.(type) {
			case float64:
				timeoutMs = int(v)
			case int:
				timeoutMs = v
			}
		}

		env := t.execute(ctx, contracts_phase3.RunScriptArgs{
			Interpreter: interpreter,
			Script:      script,
			WorkingDir:  workingDir,
			TimeoutMs:   timeoutMs,
		})

		resBytes, err := json.Marshal(env)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resBytes)), nil

	})
}

func (t *runScriptTool) execute(ctx context.Context, args contracts_phase3.RunScriptArgs) contracts.Envelope {
	startTime := time.Now()
	meta := contracts.Meta{}

	if args.Interpreter != "bash" && args.Interpreter != "sh" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "interpreter must be 'bash' or 'sh'", false, meta)
	}

	if strings.TrimSpace(args.Script) == "" {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, "script is required", false, meta)
	}

	// Policy Gate Check
	var err error
	if args.WorkingDir != "" {
		err = t.policy.Evaluate(ctx, t.Name(), []string{args.WorkingDir}, true)
	} else {
		// If working_dir is omitted, we still need to evaluate the tool name to ensure allow_run_script = true
		// Evaluate requires at least an empty string to trigger root comparison safely if we don't know the default,
		// but `run_script` default is allowedRoot. Evaluate doesn't know allowedRoot without passing it.
		// So we pass an empty slice which skips the path checks, but checks toolName.
		err = t.policy.Evaluate(ctx, t.Name(), []string{}, true)
	}
	if err != nil {
		return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, err.Error(), false, meta)
	}

	// Blocked Binaries Scan
	runScriptConfig := t.policy.RunScriptConfig()
	blockedBinaries := runScriptConfig.BlockedBinaries
	if len(blockedBinaries) == 0 {
		blockedBinaries = []string{"curl", "wget", "nc", "ncat", "netcat", "ssh", "scp", "rsync"}
	}

	// Scan the script for blocked binaries as words
	for _, bin := range blockedBinaries {
		matched, _ := regexp.MatchString(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(bin)), args.Script)
		if matched {
			return t.builder.Failure(t.Name(), contracts.ErrPolicyDenied, fmt.Sprintf("blocked binary found in script: %s", bin), false, meta)
		}
	}

	// Timeout logic
	limits := t.policy.Limits()
	timeout := limits.TimeoutMs
	if args.TimeoutMs > 0 && int64(args.TimeoutMs) <= limits.TimeoutMs {
		timeout = int64(args.TimeoutMs)
	} else if args.TimeoutMs > 0 {
		// Cannot exceed policy limits
		timeout = limits.TimeoutMs
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(execCtx, args.Interpreter)
	cmd.Stdin = strings.NewReader(args.Script)

	if args.WorkingDir != "" {
		cmd.Dir = args.WorkingDir
	} else {
		cmd.Dir = t.policy.AllowedRoot()
	}

	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + path,
		"TMPDIR=" + os.Getenv("TMPDIR"),
	}

	applySysProcAttr(cmd, runScriptConfig.AllowNetwork)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if startErr := cmd.Start(); startErr != nil {
		return t.builder.Failure(t.Name(), contracts.ErrInvalidArgs, fmt.Sprintf("failed to start script: %v", startErr), false, meta)
	}

	execErr := cmd.Wait()

	timedOut := false
	if execCtx.Err() == context.DeadlineExceeded {
		timedOut = true
	}

	exitCode := 0
	if execErr != nil {
		if exitError, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Some other error (e.g., could not start)
			exitCode = -1
		}
	}

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	maxBytes := limits.MaxOutputBytes
	totalBytes := int64(len(stdoutStr) + len(stderrStr))
	if totalBytes > maxBytes {
		meta.Truncated = true
		// Keep as much as possible, splitting maxBytes between stdout and stderr proportionally or just truncating at end
		// The prompt says "Cap combined output at max_output_bytes."
		// For simplicity, we truncate stdout and stderr so their combined length is maxBytes.
		if int64(len(stdoutStr)) >= maxBytes {
			stdoutStr = stdoutStr[:maxBytes]
			stderrStr = ""
		} else {
			stderrStr = stderrStr[:maxBytes-int64(len(stdoutStr))]
		}
	}

	meta.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	// Even if script exited non-zero, the MCP tool execution succeeded, returning the payload with ExitCode.
	return t.builder.Success(t.Name(), contracts_phase3.RunScriptData{
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, meta)
}
