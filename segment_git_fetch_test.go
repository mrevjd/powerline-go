package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
