package main

import (
	"os/user"
	"testing"
)

func testCwdPowerline(cwd, cwdMode string) *powerline {
	return &powerline{
		cwd:      cwd,
		userInfo: user.User{HomeDir: "/home/test"},
		cfg: Config{
			CwdMode:     cwdMode,
			PathAliases: AliasMap{},
		},
	}
}

func segmentPaths(segs []pathSegment) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		out[i] = s.path
	}
	return out
}

func Test_cwdToPathSegments(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		want []string
	}{
		{name: "double slash is root", cwd: "//", want: []string{"/"}},
		{name: "triple slash is root", cwd: "///", want: []string{"/"}},
		{name: "single slash is root", cwd: "/", want: []string{"/"}},
		{name: "interior double slash collapses", cwd: "/foo//bar", want: []string{"foo", "bar"}},
		{name: "normal absolute path", cwd: "/foo/bar", want: []string{"foo", "bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testCwdPowerline(tt.cwd, "fancy")
			got := cwdToPathSegments(p, tt.cwd)
			if len(got) != len(tt.want) {
				t.Fatalf("cwdToPathSegments(%q) = %v, want %v", tt.cwd, segmentPaths(got), tt.want)
			}
			for i := range got {
				if got[i].path != tt.want[i] {
					t.Errorf("segment %d = %q, want %q", i, got[i].path, tt.want[i])
				}
			}
		})
	}
}

// Regression test for #424: `cd //` used to panic. bash exports PWD="//",
// which reached cwdToPathSegments and produced zero segments; dironly mode
// then sliced pathSegments[len-1:] = [-1:] and panicked.
func Test_segmentCwd_doubleSlash_dironly_doesNotPanic(t *testing.T) {
	p := testCwdPowerline("//", "dironly")

	segs := segmentCwd(p) // must not panic

	if len(segs) != 1 {
		t.Fatalf("segmentCwd(//) in dironly = %d segments, want 1", len(segs))
	}
	if segs[0].Content != "/" {
		t.Errorf("segmentCwd(//) in dironly content = %q, want %q", segs[0].Content, "/")
	}
}
