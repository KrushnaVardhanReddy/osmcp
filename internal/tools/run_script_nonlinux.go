//go:build !linux

package tools

import (
	"os/exec"
)

func applySysProcAttr(cmd *exec.Cmd, allowNetwork bool) {
	// Not supported on this OS
}
