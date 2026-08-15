//go:build linux

package tools

import (
	"os"
	"os/exec"
	"syscall"
)

func applySysProcAttr(cmd *exec.Cmd, allowNetwork bool) {
	if !allowNetwork && os.Getuid() == 0 {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}

		cmd.SysProcAttr.Unshareflags = syscall.CLONE_NEWNET
	}
}
