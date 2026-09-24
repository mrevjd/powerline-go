package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMain dispatches to the background-fetch supervisor when re-exec'd,
// mirroring main.go: under `go test`, os.Executable() is the test binary, so
// startBackgroundFetch re-execs it and needs this same dispatch to run the fetch.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == backgroundFetchArg {
		runBackgroundFetch(os.Args[2])
		return
	}
	os.Exit(m.Run())
}

func TestStartBackgroundFetch(t *testing.T) {
	dir := t.TempDir()
	run := func(wd string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = wd
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	bare := filepath.Join(dir, "bare.git")
	seed := filepath.Join(dir, "seed")
	clone := filepath.Join(dir, "clone")
	other := filepath.Join(dir, "other")

	run(dir, "init", "--bare", "--initial-branch=main", bare)
	run(dir, "clone", bare, seed)
	if err := os.WriteFile(filepath.Join(seed, "f.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(seed, "add", "f.txt")
	run(seed, "commit", "-m", "one")
	run(seed, "push", "origin", "main")

	run(dir, "clone", bare, clone)
	run(dir, "clone", bare, other)

	if err := os.WriteFile(filepath.Join(other, "f.txt"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(other, "add", "f.txt")
	run(other, "commit", "-m", "two")
	run(other, "push", "origin", "main")
	wantSHA := run(bare, "rev-parse", "main")

	t.Chdir(clone)
	commonDirOut, err := runGitCommand("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		t.Fatalf("git-common-dir: %v", err)
	}
	stamp := filepath.Join(strings.TrimSpace(commonDirOut), "powerline-go-fetch")

	startBackgroundFetch(time.Minute, time.Minute)
	if _, err := os.Stat(stamp); err != nil {
		t.Fatalf("expected stamp file to exist: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		got, _ := runGitCommand("git", "rev-parse", "origin/main")
		if strings.TrimSpace(got) == wantSHA {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("origin/main did not reach %s after fetch, got %q", wantSHA, got)
		}
		time.Sleep(100 * time.Millisecond)
	}

	now := time.Now()
	if err := os.Chtimes(stamp, now, now); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(other, "f.txt"), []byte("3"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(other, "add", "f.txt")
	run(other, "commit", "-m", "three")
	run(other, "push", "origin", "main")

	startBackgroundFetch(time.Minute, time.Minute)
	time.Sleep(time.Second)

	got, _ := runGitCommand("git", "rev-parse", "origin/main")
	if strings.TrimSpace(got) != wantSHA {
		t.Fatalf("expected throttled fetch to leave origin/main at %s, got %q", wantSHA, got)
	}
}

// TestBackgroundFetchTimeoutKillsProcessGroup verifies that when a fetch
// outlives its timeout, killFetch's SIGKILL to the process group takes down
// not just the fake git but a grandchild it spawned, since a stalled ssh or
// credential helper must not be left holding the connection open.
func TestBackgroundFetchTimeoutKillsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups are POSIX-only")
	}

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
