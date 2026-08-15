# `run_script` Specification

## Overview
`run_script` is the Tier 2 execution engine. It allows agents to run arbitrary shell scripts when
structured tools fall short. It is the highest-risk tool in the system.

## Security Constraints
- Must be explicitly allowed via `policy.allow_run_script = true`.
- Network access dropped via `syscall.CLONE_NEWNET` (Linux only) unless `policy.run_script.allow_network = true`.
- Script content scanned for blocked binaries (e.g. `curl`, `wget`).
- Executed via `exec.CommandContext` using `bash` or `sh`, passing the script via **stdin** (not a temp file).
- Environment is strictly sanitized to `HOME`, `PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`, and `TMPDIR` only.

## Output
Returns `stdout`, `stderr`, `exit_code`, and a boolean `timed_out` if the script hit the timeout limit.

---

## CRITICAL Implementation Rules (Read These Before Writing Any Code)

### Rule 1: NEVER inherit the parent's stdin/stdout/stderr

`osmcp` is an MCP stdio server — its own stdin and stdout are the JSON-RPC communication channel.
If the subprocess inherits them, bash will consume the MCP protocol stream and corrupt all further
communication. The process will be killed with a broken pipe and Go will report **exit code -1**.

**ALWAYS explicitly set all three streams:**

```go
var stdoutBuf, stderrBuf bytes.Buffer

cmd := exec.CommandContext(ctx, interpreter)
cmd.Stdin  = strings.NewReader(script)   // script body via stdin pipe — NOT inherited
cmd.Stdout = &stdoutBuf                  // captured — NOT os.Stdout
cmd.Stderr = &stderrBuf                  // captured — NOT os.Stderr
```

**Never do:**
```go
// BAD: inherits MCP server's stdin/stdout — WILL corrupt the protocol
cmd := exec.CommandContext(ctx, interpreter, "-c", script)
// (no Stdin/Stdout/Stderr set)
```

### Rule 2: Distinguish Start failure from runtime failure

`cmd.ProcessState` is `nil` if `cmd.Start()` fails (e.g., interpreter not found in PATH).
Calling `cmd.ProcessState.ExitCode()` on a nil pointer causes a panic. Always check `cmd.Start()`
error before calling `cmd.Wait()`:

```go
if err := cmd.Start(); err != nil {
    // Interpreter not found, permission denied, etc.
    // Return EXEC_FAILED envelope — do NOT try to read ExitCode
    return execFailedEnvelope(err.Error()), nil
}
waitErr := cmd.Wait()
```

Then after `cmd.Wait()`, distinguish exit code from signal kill:

```go
var exitErr *exec.ExitError
switch {
case waitErr == nil:
    exitCode = 0
case errors.As(waitErr, &exitErr):
    exitCode = exitErr.ExitCode() // script ran, exited non-zero — normal
default:
    // killed by signal (e.g., SIGKILL from context timeout)
    // Check ctx.Err() to determine if this is a timeout
    timedOut = ctx.Err() == context.DeadlineExceeded
    exitCode = -1
}
```

### Rule 3: CLONE_NEWNET requires root — skip it in tests

`syscall.CLONE_NEWNET` requires `CAP_SYS_ADMIN` on Linux. In the Jules sandbox, the test runner
does **not** have this capability. Guard it with a build tag or a runtime capability check:

```go
// Only attempt CLONE_NEWNET if running as root or if the env flag is not set
if policy.AllowNetwork == false && os.Getuid() == 0 {
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Unshareflags: syscall.CLONE_NEWNET,
    }
}
```

Or better: wrap it in a `//go:build linux` file so it does not even compile on non-Linux.

### Rule 4: PATH must be set explicitly in the sanitized environment

If you strip the environment to just `HOME` and `TMPDIR`, `bash` cannot find any binaries
(including built-ins that call external commands). Always include a safe, explicit PATH:

```go
env := []string{
    "HOME=" + os.Getenv("HOME"),
    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    "TMPDIR=" + os.TempDir(),
}
cmd.Env = env
```

### Rule 5: E2E test must use a policy with allow_run_script = true

The E2E test policy file must explicitly set `allow_run_script = true` in the `[policy]` section.
Without this, the tool returns POLICY_DENIED before any execution happens.

```toml
[policy]
allowed_root     = "/tmp/osmcp_test"
allow_mutation   = true
allow_run_script = true
allowed_tools    = ["run_script"]

[limits]
timeout_ms       = 10000
max_output_bytes = 524288
max_matches      = 200

[audit]
destination = "stderr"
```
