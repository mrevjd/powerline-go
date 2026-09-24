//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detachProcess puts cmd in its own session. With no controlling terminal, ssh
// and git cannot open /dev/tty to ask for a passphrase or password.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
