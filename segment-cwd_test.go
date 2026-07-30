package main

import (
	"fmt"
	"os/user"
	"reflect"
	"strings"
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

// The alias matcher slides a window of the key's length along the path segments.
// Its bound used to shrink as the window advanced, so a window was only found in
// the first half of the path and an alias covering the final component never
// matched unless it started at index 0. This walks every window size at every
// offset over a fixed path, which is where that bug lives: below the cwd modes,
// so it covers the segmented modes as well as plain.
func Test_maybeAliasPathSegments_everyWindow(t *testing.T) {
	const cwd = "/a/b/c/d"
	segments := []string{"a", "b", "c", "d"}

	for size := 1; size <= len(segments); size++ {
		for offset := 0; offset+size <= len(segments); offset++ {
			key := strings.Join(segments[offset:offset+size], "/")

			want := make([]string, 0, len(segments))
			want = append(want, segments[:offset]...)
			want = append(want, "@X")
			want = append(want, segments[offset+size:]...)

			t.Run(fmt.Sprintf("size=%d offset=%d key=%s", size, offset, key), func(t *testing.T) {
				t.Setenv("HOME", "/home/test")
				p := testCwdPowerline(cwd, "fancy")
				p.cfg.PathAliases = AliasMap{key: "@X"}

				got := segmentPaths(cwdToPathSegments(p, cwd))

				if !reflect.DeepEqual(got, want) {
					t.Errorf("alias %q on %s = %v, want %v", key, cwd, got, want)
				}
			})
		}
	}
}

// Two aliases of equal length competing for the same path must resolve the same
// way every time. Sorting the keys on length alone left equal-length keys in Go's
// randomised map iteration order, so the same cwd rendered a different prompt
// between invocations. The tie is broken lexicographically, so "app/lib" wins.
func Test_maybeAliasPathSegments_equalLengthKeysAreDeterministic(t *testing.T) {
	const cwd = "/src/app/lib"
	want := []string{"src", "@L"}

	for i := 0; i < 50; i++ {
		t.Setenv("HOME", "/home/test")
		p := testCwdPowerline(cwd, "fancy")
		p.cfg.PathAliases = AliasMap{"src/app": "@A", "app/lib": "@L"}

		got := segmentPaths(cwdToPathSegments(p, cwd))

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: %s = %v, want %v (equal-length keys must not depend on map order)", i, cwd, got, want)
		}
	}
}

// Plain mode shares cwdToPathSegments with the segmented modes, so home
// abbreviation, path normalisation and -path-aliases cannot drift between them.
// The absolute leading separator that cwdToPathSegments drops has to survive the
// round trip.
func Test_segmentCwd_plain(t *testing.T) {
	tests := []struct {
		name    string
		cwd     string
		aliases AliasMap
		want    string
	}{
		{name: "absolute path keeps its leading separator", cwd: "/etc/nginx", want: "/etc/nginx"},
		{name: "root", cwd: "/", want: "/"},
		{name: "double slash normalises to root", cwd: "//", want: "/"},
		{name: "interior double slash collapses", cwd: "/etc//nginx", want: "/etc/nginx"},
		{name: "home", cwd: "/home/test", want: "~"},
		{name: "inside home", cwd: "/home/test/proj", want: "~/proj"},
		{name: "sibling of home is not home", cwd: "/home/testother", want: "/home/testother"},
		{name: "relative cwd gains no separator", cwd: "foo/bar", want: "foo/bar"},
		{
			name:    "alias replacing the leading run drops the separator",
			cwd:     "/etc/nginx",
			aliases: AliasMap{"/etc": "@E"},
			want:    "@E/nginx",
		},
		{
			name:    "alias under home",
			cwd:     "/home/test/work/x",
			aliases: AliasMap{"~/work": "@W"},
			want:    "@W/x",
		},
		{
			name:    "alias mid-path, as in the segmented modes",
			cwd:     "/etc/nginx/conf",
			aliases: AliasMap{"nginx": "@N"},
			want:    "/etc/@N/conf",
		},
		{
			name:    "alias covering the last component",
			cwd:     "/etc/nginx",
			aliases: AliasMap{"nginx": "@N"},
			want:    "/etc/@N",
		},
		{
			name:    "multi-component alias covering the last components",
			cwd:     "/proj/src/main",
			aliases: AliasMap{"src/main": "@SM"},
			want:    "/proj/@SM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", "/home/test")
			p := testCwdPowerline(tt.cwd, "plain")
			if tt.aliases != nil {
				p.cfg.PathAliases = tt.aliases
			}

			segs := segmentCwd(p)

			if len(segs) != 1 {
				t.Fatalf("segmentCwd(%q) in plain = %d segments, want 1", tt.cwd, len(segs))
			}
			if segs[0].Content != tt.want {
				t.Errorf("segmentCwd(%q) in plain = %q, want %q", tt.cwd, segs[0].Content, tt.want)
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
