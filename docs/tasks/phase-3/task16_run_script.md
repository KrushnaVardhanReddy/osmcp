# Task 16: `run_script` — Tier 2 Execution Engine

## Goal
Implement the `run_script` Tier 2 tool specified in `docs/ARCHITECTURE.md` (§ Tier 2 — `run_script`).

This is the safety valve that allows AI agents to run arbitrary shell scripts (bash/sh) when the
structured Tier 1 tools cannot cover a use case. It is the highest-risk tool in the system and
must be implemented with the tightest sandboxing of any tool.

## Spec Reference
`docs/ARCHITECTURE.md` — Tier 2 section  
`docs/specs/cross_cutting/02_policy_engine_spec.md` — `allow_run_script` flag  
`docs/specs/phase-3/02_run_script_spec.md` (to be created alongside implementation)

## Tool to Implement
- `run_script`

## Parameters
- `interpreter` (string, required): Must be one of `"bash"` or `"sh"`. No other values are accepted.
- `script` (string, required): The shell script body to execute (passed to the interpreter via stdin, **not** via a temp file on disk).
- `working_dir` (string, optional): Absolute path inside `allowed_root` to set as CWD. Defaults to `allowed_root`.
- `timeout_ms` (int, optional): Per-call override. Must be ≤ the policy `timeout_ms`. If omitted, uses the policy limit.

## Implementation Details & Context

### 1. Policy Gate (CRITICAL — hardest enforcement in the system)
```
a) Check policyEngine.Evaluate(ctx, "run_script", []string{working_dir}, true)
b) Check policy.AllowRunScript == true. If false, return POLICY_DENIED.
```
Both checks must pass. A policy with `allow_mutation = true` but `allow_run_script = false` must still block this tool.

### 2. Execution
- Use `exec.CommandContext(ctx, interpreter)` — **never** `exec.Command("bash", "-c", script)`.
- Pass the script body via `cmd.Stdin = strings.NewReader(script)`.
- Set `cmd.Dir = working_dir`.
- Set environment to a minimal safe set: `HOME`, `PATH`, `TMPDIR` only. Strip everything else.
- Enforce `timeout_ms` via the context deadline.
- Capture both stdout and stderr into separate strings. Cap combined output at `max_output_bytes`.

### 3. Blocklist
Even with `allow_run_script = true`, these binaries must be blocked if they appear in the script:
Read the blocklist from `policy.RunScript.BlockedBinaries`. Default blocklist if empty:
`["curl", "wget", "nc", "ncat", "netcat", "ssh", "scp", "rsync"]`

Scan the script text for the presence of any blocked binary name as a word (simple string scan is sufficient — no need for a full shell parser).

### 4. Network
If `policy.RunScript.AllowNetwork == false` (default), add `cmd.SysProcAttr` with `Unshareflags: syscall.CLONE_NEWNET` on Linux to drop network access from the subprocess. (Gracefully skip if running on non-Linux platforms.)

### 5. Response Payload (`RunScriptData`)
```go
type RunScriptData struct {
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    ExitCode int    `json:"exit_code"`
    TimedOut bool   `json:"timed_out"`
}
```

### 6. Location
- Implementation: `internal/tools/run_script.go`
- Tests: `internal/tools/run_script_test.go`
- Registration in: `cmd/osmcp/main.go`

## Testing
- Unit test: verify `allow_run_script = false` returns POLICY_DENIED.
- Unit test: verify a blocked binary in the script returns POLICY_DENIED.
- Unit test: verify timeout fires and `timed_out = true` in the response.
- Unit test: verify stdout and stderr are captured correctly.
- Unit test: verify output is truncated when it exceeds `max_output_bytes`.
- Unit test: verify `working_dir` outside `allowed_root` returns POLICY_DENIED.
- Integration test: run a simple `echo hello` script end-to-end via the MCP server.

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
- Any existing Phase 1 or Phase 2 tool implementations
