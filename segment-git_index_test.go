package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// -git-assume-unchanged-size has to read the repository's index wherever in
// the work tree the prompt is drawn, not only at its root.
func TestGitAssumeUnchangedSizeFromSubdirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// Enough tracked files for the index to pass the 1 KB threshold.
	for i := 0; i < 40; i++ {
		if err := os.WriteFile(filepath.Join(sub, fmt.Sprintf("file-%02d.txt", i)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-q", "-m", "init")
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	p := &powerline{
		cwd:     sub,
		cfg:     Config{GitAssumeUnchangedSize: 1, GitMode: "fancy"},
		symbols: SymbolTemplate{RepoUntracked: "U"},
	}
	for _, seg := range segmentGit(p) {
		if seg.Name == "git-status" && strings.HasSuffix(seg.Content, "U") {
			t.Errorf("untracked files reported (%q) although the index is over the size limit", seg.Content)
		}
	}
}
