# Task 17: `sort` Tool

## Goal
Implement the `sort` text-processing tool specified in `docs/specs/phase-2/04_sort_spec.md`.

## Spec Reference
`docs/specs/phase-2/04_sort_spec.md`

## Contract
Implement the `SortTool` interface defined in `docs/contracts/phase-2/sort_contract.go`.
Input shape: `SortArgs`. Output shape: `SortData` (placed in `Envelope.Data` on success).

## Tool to Implement
- `sort`

## Implementation Details & Context

1. **Read-only tool** — `IsMutating()` returns `false`.
2. **Policy Check**: Call `policyEngine.Evaluate(ctx, "sort", []string{args.Path}, false)`.
3. **Implementation**: Use only Go stdlib — read the file with `os.Open`, split by newlines, sort with `sort.Strings()` (or `sort.Slice` for numeric), deduplicate if `unique=true`, return the result. **No subprocess execution.**
4. **Numeric sort**: Use `strconv.ParseFloat` to convert each line before comparison. Lines that fail to parse sort lexicographically after numeric lines.
5. **Output data shape**:
   ```go
   type SortData struct {
       Lines []string `json:"lines"`
       Count int      `json:"count"`
   }
   ```
6. **Truncation**: If the total output size exceeds `policy.Limits().MaxOutputBytes`, stop appending lines and set `meta.Truncated = true`.
7. **Location**: `internal/tools/sort.go` and `internal/tools/sort_test.go`.
8. **Registration**: Implement `RegisterMCP(s *server.MCPServer)` and register in `cmd/osmcp/main.go`.

## Testing
- `sort` on a plain text file returns lines in alphabetical order.
- `sort --reverse` returns lines in reverse order.
- `sort --unique` deduplicates identical adjacent lines.
- `sort --numeric` sorts `["10", "2", "1"]` as `["1", "2", "10"]`, not `["1", "10", "2"]`.
- Empty file returns `{"lines": [], "count": 0}`.
- Path outside `allowed_root` returns `POLICY_DENIED`.
- Directory path returns `INVALID_ARGS`.

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
