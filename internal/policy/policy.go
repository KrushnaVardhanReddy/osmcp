package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Policy matches the TOML schema in spec §2.
type Policy struct {
	PolicyConfig PolicySection   `toml:"policy"`
	RunScript    RunScriptSection `toml:"run_script"`
	Limits       LimitsSection   `toml:"limits"`
	Audit        AuditSection    `toml:"audit"`
}

type RunScriptSection struct {
	BlockedBinaries []string `toml:"blocked_binaries"`
	AllowNetwork    bool     `toml:"allow_network"`
}

type PolicySection struct {
	AllowedRoot    string   `toml:"allowed_root"`
	AllowedTools   []string `toml:"allowed_tools"`
	AllowMutation  bool     `toml:"allow_mutation"`
	AllowGitWrite  bool     `toml:"allow_git_write"`
	AllowRunScript bool     `toml:"allow_run_script"`
}

type LimitsSection struct {
	TimeoutMs      int64 `toml:"timeout_ms"`
	MaxOutputBytes int64 `toml:"max_output_bytes"`
	MaxMatches     int   `toml:"max_matches"`
}

type AuditSection struct {
	Destination string `toml:"destination"`
	Path        string `toml:"path"`
}

// Known tools for validation
var knownTools = map[string]bool{
	"grep": true, "find": true, "ls": true, "cat": true, "head": true, "tail": true,
	"tree": true, "du": true,
	"wc": true, "stat": true, "sort": true,
	"git_status": true, "git_diff": true, "git_log": true,
	"sed": true, "jq": true, "diff": true,
	"cp": true, "mv": true, "rm": true, "mkdir": true,
	"git_add": true, "git_commit": true, "git_checkout": true,
	"git_branch": true, "git_pull": true, "git_push": true, "patch": true,
	"awk": true,
	"tar": true,
	"run_script": true,
}

// LoadFromFile loads and parses a Policy from a TOML file.
func LoadFromFile(path string) (*Policy, error) {
	var p Policy
	if _, err := toml.DecodeFile(path, &p); err != nil {
		return nil, fmt.Errorf("failed to decode policy file: %w", err)
	}
	return &p, nil
}

// Validate checks the policy against rules in spec §6.
func Validate(p *Policy) []error {
	var errs []error

	if p.PolicyConfig.AllowedRoot == "" {
		errs = append(errs, errors.New("allowed_root is required"))
	} else if !filepath.IsAbs(p.PolicyConfig.AllowedRoot) {
		errs = append(errs, fmt.Errorf("allowed_root must be an absolute path: %s", p.PolicyConfig.AllowedRoot))
	} else {
		info, err := os.Stat(p.PolicyConfig.AllowedRoot)
		if err != nil {
			errs = append(errs, fmt.Errorf("allowed_root does not exist or is not accessible: %w", err))
		} else if !info.IsDir() {
			errs = append(errs, fmt.Errorf("allowed_root must be a directory: %s", p.PolicyConfig.AllowedRoot))
		}
	}

	for _, t := range p.PolicyConfig.AllowedTools {
		if !knownTools[t] {
			errs = append(errs, fmt.Errorf("unknown tool in allowed_tools: %s", t))
		}
	}

	if p.Limits.TimeoutMs < 100 || p.Limits.TimeoutMs > 60000 {
		errs = append(errs, fmt.Errorf("timeout_ms must be between 100 and 60000, got: %d", p.Limits.TimeoutMs))
	}

	if p.Limits.MaxOutputBytes < 1024 || p.Limits.MaxOutputBytes > 10485760 {
		errs = append(errs, fmt.Errorf("max_output_bytes must be between 1024 and 10485760, got: %d", p.Limits.MaxOutputBytes))
	}

	if p.Audit.Destination != "stderr" && p.Audit.Destination != "file" {
		errs = append(errs, fmt.Errorf("audit destination must be 'stderr' or 'file', got: %s", p.Audit.Destination))
	}

	return errs
}

// DefaultPolicy returns safe defaults (read-only, cwd as root).
func DefaultPolicy() *Policy {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	return &Policy{
		PolicyConfig: PolicySection{
			AllowedRoot: cwd,
			AllowedTools: []string{
				"grep", "find", "ls", "cat", "head", "tail",
				"tree", "du",
				"wc", "stat", "sort",
				"git_status", "git_diff", "git_log",
				"git_add", "git_commit", "git_checkout",
				"git_branch", "git_pull", "git_push",
				"patch",
				"sed", "jq", "diff", "awk", "tar",
			},
			AllowMutation:  false,
			AllowGitWrite:  false,
			AllowRunScript: false,
		},
		Limits: LimitsSection{
			TimeoutMs:      10000,
			MaxOutputBytes: 524288,
			MaxMatches:     1000,
		},
		Audit: AuditSection{
			Destination: "stderr",
		},
	}
}
