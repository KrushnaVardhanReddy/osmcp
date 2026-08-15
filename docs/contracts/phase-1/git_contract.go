// osmcp — Git Intelligence Tools Contract (Phase 1)
//
// Spec reference: docs/specs/phase-1/03_git_tools_spec.md
// Library: github.com/go-git/go-git/v5 (Pure Go — no os/exec)
// Prerequisite: docs/contracts/cross_cutting/core_contracts.go must be complete.

package contracts

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// git_status (Working Tree Status)
// ─────────────────────────────────────────────────────────────────────────────

type GitStatusArgs struct {
	Path string `json:"path" jsonschema:"required,description=Absolute path to the repository root or any subdirectory."`
}

type GitStatusData struct {
	Branch     string           `json:"branch"`
	HeadCommit string           `json:"head_commit"`
	Clean      bool             `json:"clean"`
	Staged     []GitFileStatus  `json:"staged"`
	Unstaged   []GitFileStatus  `json:"unstaged"`
	Untracked  []string         `json:"untracked"`
}

type GitFileStatus struct {
	Path string `json:"path"`
	Code string `json:"code"` // "A", "M", "D", "R", "?"
}

// ─────────────────────────────────────────────────────────────────────────────
// git_diff (File or Commit Diff)
// ─────────────────────────────────────────────────────────────────────────────

type GitDiffArgs struct {
	Path       string  `json:"path" jsonschema:"required,description=Absolute path to the repository root."`
	File       *string `json:"file,omitempty" jsonschema:"description=Optional relative file path to restrict the diff."`
	FromCommit *string `json:"from_commit,omitempty" jsonschema:"description=Starting commit hash. Defaults to HEAD~1."`
	ToCommit   *string `json:"to_commit,omitempty" jsonschema:"description=Ending commit hash. Defaults to HEAD."`
}

type GitDiffData struct {
	Patches         []GitPatch `json:"patches"`
	TotalAdditions  int        `json:"total_additions"`
	TotalDeletions  int        `json:"total_deletions"`
}

type GitPatch struct {
	File      string `json:"file"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Diff      string `json:"diff"`
}

// ─────────────────────────────────────────────────────────────────────────────
// git_log (Commit History)
// ─────────────────────────────────────────────────────────────────────────────

type GitLogArgs struct {
	Path       string  `json:"path" jsonschema:"required,description=Absolute path to the repository root."`
	MaxCommits int     `json:"max_commits" jsonschema:"default=20,minimum=1,maximum=200,description=Maximum number of commits to return."`
	File       *string `json:"file,omitempty" jsonschema:"description=Optional file path to filter commits by."`
	Branch     *string `json:"branch,omitempty" jsonschema:"description=Branch to read history from. Defaults to current HEAD."`
}

type GitLogData struct {
	Commits []GitCommit `json:"commits"`
	Count   int         `json:"count"`
}

type GitCommit struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Date      time.Time `json:"date"`
	Message   string    `json:"message"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariants (verified by unit tests)
// ─────────────────────────────────────────────────────────────────────────────

// INV-GIT-01: Pure Go — os/exec is never called; only go-git library is used.
// INV-GIT-02: Policy Check — Path outside AllowedRoot returns ok=false, POLICY_DENIED.
// INV-GIT-03: Non-Repo — Path not inside a git repo returns ok=false, EXEC_FAILED.
// INV-GIT-04: Clean Repo — git_status on a clean repo returns clean=true, empty arrays.
// INV-GIT-05: Invalid Commit — Unknown from_commit hash returns ok=false, INVALID_ARGS.
// INV-GIT-06: Diff Truncation — Patches stop when max_output_bytes is exceeded (meta.Truncated=true).
// INV-GIT-07: Log Limit — max_commits is capped at min(args.MaxCommits, policy.MaxMatches).
// INV-GIT-08: Read-Only — No git operation (commit, push, reset) is ever performed.
