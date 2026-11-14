package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// Pins how each -git-mode renders a repository with one staged, one modified
// and one untracked file.
func TestSegmentGitModes(t *testing.T) {
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
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("tracked", "a")
	git("add", "tracked")
	git("commit", "-q", "-m", "init")
	write("tracked", "b")
	write("staged", "a")
	git("add", "staged")
	write("untracked", "a")
	t.Chdir(root)

	symbols := SymbolTemplate{RepoBranch: "B", RepoStaged: "S", RepoNotStaged: "N", RepoUntracked: "U", RepoStashed: "$"}
	theme := Theme{RepoDirtyFg: 1, RepoDirtyBg: 2, GitStagedFg: 3, GitStagedBg: 4, GitNotStagedFg: 5, GitNotStagedBg: 6, GitUntrackedFg: 7, GitUntrackedBg: 8}

	type rendered struct {
		Name    string
		Content string
		Fg, Bg  uint8
	}
	tests := []struct {
		mode string
		want []rendered
	}{
		{"fancy", []rendered{
			{"git-branch", "B main", 1, 2},
			{"git-status", "1S", 3, 4},
			{"git-status", "1N", 5, 6},
			{"git-status", "1U", 7, 8},
		}},
		{"compact", []rendered{{"git-branch", "B main 1S 1N 1U", 1, 2}}},
		{"simple", []rendered{{"git-branch", "B main SNU", 1, 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			p := &powerline{cwd: root, cfg: Config{GitMode: tt.mode}, symbols: symbols, theme: theme}
			var got []rendered
			for _, s := range segmentGit(p) {
				got = append(got, rendered{s.Name, s.Content, s.Foreground, s.Background})
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
