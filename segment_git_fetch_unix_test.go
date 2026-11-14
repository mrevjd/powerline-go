//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestBackgroundFetchTimeoutKillsProcessGroup verifies that when a fetch
// outlives its timeout, killFetch's SIGKILL to the process group takes down
// not just the fake git but a grandchild it spawned, since a stalled ssh or
// credential helper must not be left holding the connection open.
func TestBackgroundFetchTimeoutKillsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	t.Chdir(dir)

	fakeDir := t.TempDir()
	pidDir := t.TempDir()
	gitPidFile := filepath.Join(pidDir, "git.pid")
	childPidFile := filepath.Join(pidDir, "child.pid")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not found")
	}
	// Only the fetch stalls: exec.Command resolves "git" through this
	// process's PATH, so the stamp's rev-parse reaches the fake too.
	script := fmt.Sprintf("#!/bin/sh\ncase \" $* \" in *\" fetch \"*) ;; *) exec %s \"$@\" ;; esac\nsleep 60 &\necho $! > %s\necho $$ > %s\nwait\n", realGit, childPidFile, gitPidFile)
	if err := os.WriteFile(filepath.Join(fakeDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// The supervisor inherits os.Environ, so the re-exec'd background fetch
	// picks up this fake git ahead of the real one.
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))

	alive := func(pid int) bool {
		return syscall.Kill(pid, 0) == nil
	}
	readPID := func(path string) (int, bool) {
		b, err := os.ReadFile(path)
		if err != nil {
			return 0, false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		return pid, err == nil
	}

	startBackgroundFetch(2*time.Second, 2*time.Second)

	var gitPID, childPID int
	deadline := time.Now().Add(5 * time.Second)
	for {
		gp, gok := readPID(gitPidFile)
		cp, cok := readPID(childPidFile)
		if gok && cok {
			gitPID, childPID = gp, cp
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for fake git and child pid files")
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Cleanup(func() {
		for _, pid := range []int{gitPID, childPID} {
			if pid > 0 {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	time.Sleep(time.Second)
	if !alive(gitPID) {
		dbg, _ := exec.Command("ps", "-eo", "pid,ppid,pgid,sid,stat,cmd").CombinedOutput()
		t.Fatalf("fake git (pid %d) exited before the timeout fired\n%s", gitPID, dbg)
	}

	deadline = time.Now().Add(10 * time.Second)
	for alive(gitPID) || alive(childPID) {
		if time.Now().After(deadline) {
			t.Fatalf("expected timeout to kill fake git (pid %d, alive=%v) and its child (pid %d, alive=%v)", gitPID, alive(gitPID), childPID, alive(childPID))
		}
		time.Sleep(100 * time.Millisecond)
	}
}
