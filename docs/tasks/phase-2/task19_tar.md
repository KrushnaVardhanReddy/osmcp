# Task 19: `tar` Tool

## Goal
Implement the `tar` read-only archive inspection tool specified in `docs/specs/phase-2/06_tar_spec.md`.

## Spec Reference
`docs/specs/phase-2/06_tar_spec.md`

## Contract
Implement the `TarTool` interface defined in `docs/contracts/phase-2/tar_contract.go`.
Input shape: `TarArgs`. Output shape: `TarListData` (action=list) or `TarExtractData` (action=extract),
both placed in `Envelope.Data` on success.

## Tool to Implement
- `tar`

## Implementation Details & Context

1. **Read-only tool** — `IsMutating()` returns `false`. This tool NEVER writes files to disk.
2. **Policy Check**: Call `policyEngine.Evaluate(ctx, "tar", []string{args.Path}, false)`.
3. **Pure Go**: Use Go stdlib `archive/tar` + `compress/gzip`, `compress/bzip2`, and `github.com/ulikunitz/xz` for decompression. **Never call `exec.Command("tar", ...)`.**
4. **Format detection**: Auto-detect compression format from file extension:
   - `.tar` → raw
   - `.tar.gz`, `.tgz` → `compress/gzip`
   - `.tar.bz2` → `compress/bzip2`
   - `.tar.xz` → `github.com/ulikunitz/xz`
5. **Path traversal protection**: For every entry in the archive, check that the `Name` field does NOT start with `/` or contain `../`. Reject the entire archive with `EXEC_FAILED` if any such entry is found.
6. **Action `"list"` data shape**:
   ```go
   type TarEntry struct {
       Name  string `json:"name"`
       Size  int64  `json:"size"`
       Mode  string `json:"mode"`
       IsDir bool   `json:"is_dir"`
   }
   type TarListData struct {
       Entries []TarEntry `json:"entries"`
       Count   int        `json:"count"`
   }
   ```
7. **Action `"extract"` data shape**:
   ```go
   type TarExtractData struct {
       Entry   string `json:"entry"`
       Content string `json:"content"`
       Size    int64  `json:"size"`
   }
   ```
   - Read the matched entry into memory (up to `max_output_bytes`).
   - Validate the content is valid UTF-8 with `utf8.Valid()`. If not, return `EXEC_FAILED` with `"binary file cannot be extracted as text"`.
8. **Truncation**: For `list`, stop adding entries at `max_matches` and set `meta.Truncated = true`. For `extract`, cap at `max_output_bytes`.
9. **Location**: `internal/tools/tar.go` and `internal/tools/tar_test.go`.
10. **Registration**: Implement `RegisterMCP(s *server.MCPServer)` and register in `cmd/osmcp/main.go`.

## Testing
- `tar list` on a `.tar.gz` returns correct entry metadata.
- `tar list` on a `.tar.bz2` decompresses and lists correctly.
- `tar extract` returns the content of a known text entry.
- `tar extract` on a binary entry returns `EXEC_FAILED`.
- An archive with a `../../etc/passwd` entry returns `EXEC_FAILED` (path traversal protection).
- Path outside `allowed_root` returns `POLICY_DENIED`.
- Non-existent archive returns `NOT_FOUND`.

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
