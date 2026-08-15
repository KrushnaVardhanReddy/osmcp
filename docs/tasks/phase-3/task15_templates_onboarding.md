# Task 15: Templates & Agent Onboarding

## Goal
Implement the templates and `--init` scaffolding command specified in
`docs/specs/phase-3/01_templates_and_onboarding_spec.md`.

This task makes osmcp "batteries-included" — an agent or user can be productive
in under 5 minutes without writing a single config file from scratch.

## Spec Reference
`docs/specs/phase-3/01_templates_and_onboarding_spec.md`

## Deliverables

### 1. Policy Templates (`templates/policies/`)
Create four pre-built `policy.toml` files matching the spec exactly:
- `templates/policies/read-only.toml`
- `templates/policies/dev-agent.toml`
- `templates/policies/ci-agent.toml`
- `templates/policies/review-agent.toml`

### 2. System Prompt Templates (`templates/system-prompts/`)
Create two Markdown system prompt files:
- `templates/system-prompts/full-toolset.md` — All Phase 1 + Phase 2 tools with parameter
  descriptions, example JSON responses, error codes, and workflow examples.
- `templates/system-prompts/read-only.md` — Trimmed version for read/inspect-only agents.

### 3. Example Response JSON (`templates/examples/`)
Create one success + one error JSON for every tool. Each file must be a valid
JSON document that deserializes into the `Envelope` struct from
`docs/contracts/cross_cutting/core_contracts.go`. Tools to cover:
- Phase 1: `grep`, `ls`, `cat` (include a `_truncated` variant), `stat`, `wc`,
  `head`, `tail`, `tree`, `du`, `find`, `git_status`, `git_diff`, `git_log`,
  `jq`, `sed`, `diff`
- Phase 2: `write_file`, `append_file`, `mkdir`, `rm`, `mv`, `cp`,
  `git_add`, `git_commit`, `git_checkout`, `git_branch`, `git_pull`, `git_push`,
  `patch`

### 4. `--init` Flag on the `osmcp` binary (`cmd/osmcp/main.go`)
Add a `--init` / `--profile` flag pair that scaffolds a starter config.

**Implementation Details:**
1. Parse `--init` and `--profile <name>` from `os.Args` using the existing `flag` package.
2. Valid profile names: `read-only`, `dev-agent`, `ci-agent`, `review-agent`.
3. Default profile if `--profile` is omitted: `dev-agent`.
4. Create `.osmcp/` in CWD if it does not exist.
5. Embed the template files using Go's `embed.FS` so the binary is fully self-contained (no external files needed at runtime).
6. Copy the selected policy template to `.osmcp/policy.toml`.
7. If `.osmcp/policy.toml` already exists, print an error message and exit with code 1 (do NOT overwrite).
8. On success, print:
   ```
   ✅ Initialized osmcp with profile: <profile>
      Config: .osmcp/policy.toml
      Run:    osmcp --policy .osmcp/policy.toml
   ```

## Testing
- Unit test: verify that `--init` creates the correct file with the correct profile content.
- Unit test: verify that running `--init` twice on the same directory exits with code 1 and does not overwrite the file.
- Unit test: verify that all four profile templates produce valid TOML.
- Unit test: verify all example JSON files deserialize into `contracts.Envelope` without error.

## Location
- Templates: `templates/` (top-level directory in the repo)
- `--init` logic: `cmd/osmcp/main.go` or a new `cmd/osmcp/init.go`
- Tests: `cmd/osmcp/init_test.go`

## Do NOT modify
- `docs/contracts/` — read-only documentation
- `docs/specs/` — read-only documentation
- Any existing Phase 1 or Phase 2 tool implementations
