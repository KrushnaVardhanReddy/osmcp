package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
)

type engine struct {
	policy *Policy
	audit  contracts.AuditLogger
}

// NewEngine creates a new PolicyEngine using the given Policy and AuditLogger.
func NewEngine(p *Policy, audit contracts.AuditLogger) contracts.PolicyEngine {
	return &engine{
		policy: p,
		audit:  audit,
	}
}

// resolveForPolicy resolves symlinks on the deepest existing ancestor.
// Safe for paths that don't yet exist (e.g. file-create tools in Phase 2).
func resolveForPolicy(p string) (string, error) {
	// Fast path: path exists, resolve fully.
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}

	// Slow path: walk up to the deepest existing parent.
	dir, suffix := p, ""
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached FS root without finding anything — last resort.
			return filepath.Abs(p)
		}
		if _, err := os.Lstat(dir); err == nil {
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return "", err
			}
			if suffix == "" {
				return resolved, nil
			}
			return filepath.Join(resolved, suffix), nil
		}
		suffix = filepath.Join(filepath.Base(dir), suffix)
		dir = parent
	}
}

func (e *engine) Evaluate(ctx context.Context, toolName string, pathArgs []string, isMutating bool) error {
	var err error
	var errCode string
	startTime := time.Now()

	defer func() {
		decision := "ALLOW"
		if err != nil {
			decision = "DENIED"
		}

		if e.audit != nil {
			entry := contracts.AuditEntry{
				Timestamp:      time.Now(),
				Tool:           toolName,
				PathArgs:       pathArgs,
				PolicyDecision: decision,
				DenialCode:     errCode,
				DurationMs:     time.Since(startTime).Milliseconds(),
				OK:             err == nil,
			}
			e.audit.Log(entry)
		}
	}()

	if !e.IsToolVisible(toolName) {
		errCode = string(contracts.ErrPolicyDenied)
		err = &contracts.PolicyError{Reason: fmt.Sprintf("tool not in allowlist: %s", toolName)}
		return err
	}

		if toolName == "run_script" && !e.policy.PolicyConfig.AllowRunScript {
		errCode = string(contracts.ErrPolicyDenied)
		err = &contracts.PolicyError{Reason: "run_script not permitted by policy"}
		return err
	}

	allowedRoot := filepath.Clean(e.policy.PolicyConfig.AllowedRoot)

	for _, p := range pathArgs {
		// a. Resolve symlinks → absPath (Safely handling non-existent files)
		absPath, resolveErr := resolveForPolicy(p)
		if resolveErr != nil {
			// If we fail to resolve, fail closed.
			errCode = string(contracts.ErrPolicyDenied)
			err = &contracts.PolicyError{Reason: fmt.Sprintf("failed to resolve path %s: %v", p, resolveErr)}
			return err
		}

		absPath = filepath.Clean(absPath)

		// b. Does absPath have policy.AllowedRoot as a prefix?
		if !strings.HasPrefix(absPath, allowedRoot) || (len(absPath) > len(allowedRoot) && absPath[len(allowedRoot)] != filepath.Separator) {
			// exact match or prefix + separator
			if absPath != allowedRoot {
				errCode = string(contracts.ErrPolicyDenied)
				err = &contracts.PolicyError{Reason: fmt.Sprintf("path outside allowed root: %s", absPath)}
				return err
			}
		}
	}

	if isMutating && !e.policy.PolicyConfig.AllowMutation {
		errCode = string(contracts.ErrPolicyDenied)
		err = &contracts.PolicyError{Reason: "mutation not permitted by policy"}
		return err
	}

	if isGitWrite(toolName) && !e.policy.PolicyConfig.AllowGitWrite {
		errCode = string(contracts.ErrPolicyDenied)
		err = &contracts.PolicyError{Reason: "git write not permitted by policy"}
		return err
	}

	return nil
}

func (e *engine) IsToolVisible(toolName string) bool {
	for _, t := range e.policy.PolicyConfig.AllowedTools {
		if t == toolName {
			return true
		}
	}
	return false
}

func (e *engine) Limits() contracts.PolicyLimits {
	return contracts.PolicyLimits{
		TimeoutMs:      e.policy.Limits.TimeoutMs,
		MaxOutputBytes: e.policy.Limits.MaxOutputBytes,
		MaxMatches:     e.policy.Limits.MaxMatches,
	}
}

func (e *engine) RunScriptConfig() contracts.RunScriptConfig {
	return contracts.RunScriptConfig{
		BlockedBinaries: e.policy.RunScript.BlockedBinaries,
		AllowNetwork:    e.policy.RunScript.AllowNetwork,
	}
}

func isGitWrite(toolName string) bool {
	switch toolName {
	case "git_add", "git_commit", "git_branch":
		return true
	}
	return false
}

func (e *engine) AllowedRoot() string {
	return e.policy.PolicyConfig.AllowedRoot
}
