# Task 18: `awk` Tool

## Goal
Implement the `awk` text-processing tool specified in `docs/specs/phase-2/05_awk_spec.md`.

## Spec Reference
`docs/specs/phase-2/05_awk_spec.md`

## Contract
Implement the `AwkTool` interface defined in `docs/contracts/phase-2/awk_contract.go`.
Input shape: `AwkArgs`. Output shape: `AwkData` (placed in `Envelope.Data` on success).

## Tool to Implement
- `awk`

## Implementation Details & Context

1. **Read-only tool** — `IsMutating()` returns `false`.
2. **Policy Check**: Call `policyEngine.Evaluate(ctx, "awk", []string{args.Path}, false)`.
3. **Pure Go AWK**: Use `github.com/benhoyt/goawk`. Add it to `go.mod` if not already present. **Never call `exec.Command("awk", ...)`.**
4. **Field separator**: Pass `args.FieldSeparator` via `goawk.Config{FS: args.FieldSeparator}`.
5. **Security — block file writes**: Before executing, scan `args.Program` for the `>` redirect operator followed by a path string. If found, return `EXEC_FAILED` with message `"AWK write redirects are not permitted"`.
6. **Input**: Open the file with `os.Open`, pass the reader to goawk as stdin.
7. **Output data shape**:
   ```go
   type AwkData struct {
       Output         string `json:"output"`
       LinesProcessed int    `json:"lines_processed"`
   }
   ```
8. **Truncation**: If output exceeds `policy.Limits().MaxOutputBytes`, truncate and set `meta.Truncated = true`.
9. **Location**: `internal/tools/awk.go` and `internal/tools/awk_test.go`.
10. **Registration**: Implement `RegisterMCP(s *server.MCPServer)` and register in `cmd/osmcp/main.go`.

## Testing
- `awk '{print $2}' file.txt` correctly extracts column 2 from a whitespace-delimited file.
- Custom field separator (`-F ","`) correctly parses a CSV-like file.
- An AWK syntax error returns `EXEC_FAILED`.
- An AWK program with `> "/etc/passwd"` returns `EXEC_FAILED`.
- Path outside `allowed_root` returns `POLICY_DENIED`.
- Empty file returns `{"output": "", "lines_processed": 0}`.

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
