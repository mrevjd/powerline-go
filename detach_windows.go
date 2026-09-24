package main

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// detachProcess keeps cmd off the terminal's console and out of its process
// group, so it neither flashes a window nor dies with the shell's Ctrl+C.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}

// killFetch kills the process git was started as. With Git for Windows that is
// a launcher, so the real git and any ssh it started may outlive a timed-out
// fetch.
func killFetch(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
