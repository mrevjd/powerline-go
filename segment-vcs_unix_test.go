//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	pwl "github.com/justjanne/powerline-go/powerline"
)

// fakeTool puts an executable named name on PATH that answers each first
// argument with the matching canned output.
func fakeTool(t *testing.T, name string, outputs map[string]string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncase \"$1\" in\n"
	for arg, out := range outputs {
		script += arg + ") printf '%s' '" + out + "' ;;\n"
	}
	script += "esac\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

type segmentView struct {
	Name    string
	Content string
	Fg, Bg  uint8
}

func view(segment func(*powerline) []pwl.Segment) func(*powerline) []segmentView {
	return func(p *powerline) []segmentView {
		var views []segmentView
		for _, s := range segment(p) {
			views = append(views, segmentView{s.Name, s.Content, s.Foreground, s.Background})
		}
		return views
	}
}

func TestVcsSegments(t *testing.T) {
	p := &powerline{theme: Theme{RepoCleanFg: 1, RepoCleanBg: 2, RepoDirtyFg: 3, RepoDirtyBg: 4}}
	cases := []struct {
		name    string
		tool    string
		outputs map[string]string
		run     func(*powerline) []segmentView
		want    string
		dirty   bool
	}{
		{"hg clean", "hg", map[string]string{"branch": "default\n"}, view(segmentHg), "default", false},
		{"hg untracked and missing", "hg", map[string]string{"branch": "default\n", "status": "? new\n! gone\n"}, view(segmentHg), "default +!", true},
		{"hg modified only", "hg", map[string]string{"branch": "default\n", "status": "M file\n"}, view(segmentHg), "default ", true},
		{"bzr untracked", "bzr", map[string]string{"nick": "trunk\n", "status": "unknown:\n  new\n"}, view(segmentBzr), "trunk +", true},
		{"fossil missing", "fossil", map[string]string{"branch": "trunk\n", "changes": "MISSING gone\n"}, view(segmentFossil), "trunk !", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeTool(t, tc.tool, tc.outputs)
			got := tc.run(p)
			if len(got) != 1 {
				t.Fatalf("got %d segments, want 1: %+v", len(got), got)
			}
			wantFg, wantBg := uint8(1), uint8(2)
			if tc.dirty {
				wantFg, wantBg = 3, 4
			}
			if got[0] != (segmentView{tc.tool, tc.want, wantFg, wantBg}) {
				t.Errorf("got %+v, want {%s %q %d %d}", got[0], tc.tool, tc.want, wantFg, wantBg)
			}
		})
	}

	t.Run("no branch, no segment", func(t *testing.T) {
		fakeTool(t, "hg", map[string]string{})
		if got := segmentHg(p); len(got) != 0 {
			t.Errorf("got %+v, want none", got)
		}
	})
}

func TestReadFileUpwards(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", ".ruby-version"), []byte(" 3.3.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := readFileUpwards(deep, ".ruby-version"); !ok || got != "3.3.0" {
		t.Errorf("readFileUpwards = %q, %v; want \"3.3.0\", true", got, ok)
	}
	if got, ok := readFileUpwards(deep, ".no-such-version-file"); ok {
		t.Errorf("readFileUpwards found %q for a missing file", got)
	}
}
