//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// detachProcess puts cmd in its own session. With no controlling terminal, ssh
// and git cannot open /dev/tty to ask for a passphrase or password.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// killFetch kills the fetch's whole process group, not just git: the ssh or
// credential helper git started would otherwise hold the stalled connection
// open. The group is this supervisor's own session, so it goes too, which is
// fine because the fetch was its only job. setsid made the pgid equal the pid;
// naming the group by pid means a supervisor run without setsid fails with
// ESRCH instead of killing its caller's group.
func killFetch(*exec.Cmd) error {
	return syscall.Kill(-os.Getpid(), syscall.SIGKILL)
}
