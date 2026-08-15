# `run_script` Specification

## Overview
`run_script` is the Tier 2 execution engine. It allows agents to run arbitrary shell scripts when structured tools fall short. It is the highest-risk tool in the system.

## Security Constraints
- Must be explicitly allowed via `policy.allow_run_script = true`.
- Network access dropped via `syscall.CLONE_NEWNET` (Linux only) unless `policy.run_script.allow_network = true`.
- Script content scanned for blocked binaries (e.g. `curl`, `wget`).
- Executed via `exec.CommandContext` using `bash` or `sh`, passing the script via `stdin`.
- Environment is strictly sanitized to `HOME`, `PATH`, and `TMPDIR`.

## Output
Returns `stdout`, `stderr`, `exit_code`, and a boolean `timed_out` if the script hit the timeout limit.
