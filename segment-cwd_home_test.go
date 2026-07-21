package main

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func pathSegmentPaths(segs []pathSegment) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		out[i] = s.path
	}
	return out
}

func homePowerline(homeDir string) *powerline {
	return &powerline{
		userInfo: user.User{HomeDir: homeDir},
		cfg:      Config{CwdMode: "fancy", PathAliases: AliasMap{}},
	}
}

// Covers #418: the home directory must abbreviate to "~" even when the shell's
// $HOME differs from the passwd home directory, or when one side is a symlink
// (both common on WSL).
func Test_homeRelativePath(t *testing.T) {
	assert := func(t *testing.T, got []pathSegment, want ...string) {
		t.Helper()
		gotPaths := pathSegmentPaths(got)
		if len(gotPaths) != len(want) {
			t.Fatalf("segments = %v, want %v", gotPaths, want)
		}
		for i := range want {
			if gotPaths[i] != want[i] {
				t.Fatalf("segments = %v, want %v", gotPaths, want)
			}
		}
	}

	t.Run("cwd equals passwd home", func(t *testing.T) {
		t.Setenv("HOME", "")
		p := homePowerline("/home/passwd")
		assert(t, cwdToPathSegments(p, "/home/passwd"), "~")
	})

	t.Run("cwd inside passwd home", func(t *testing.T) {
		t.Setenv("HOME", "")
		p := homePowerline("/home/passwd")
		assert(t, cwdToPathSegments(p, "/home/passwd/proj"), "~", "proj")
	})

	t.Run("cwd equals $HOME when passwd home differs", func(t *testing.T) {
		t.Setenv("HOME", "/home/env")
		p := homePowerline("/home/passwd")
		assert(t, cwdToPathSegments(p, "/home/env/proj"), "~", "proj")
	})

	t.Run("symlinked home resolves to real cwd", func(t *testing.T) {
		base := t.TempDir()
		real := filepath.Join(base, "real")
		if err := os.Mkdir(real, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "link")
		if err := os.Symlink(real, link); err != nil {
			t.Skipf("symlinks unsupported: %v", err)
		}
		resolvedReal, err := filepath.EvalSymlinks(real)
		if err != nil {
			t.Fatal(err)
		}

		// Shell home is the symlink; os.Getwd returns the resolved path.
		t.Setenv("HOME", link)
		p := homePowerline(link)
		assert(t, cwdToPathSegments(p, filepath.Join(resolvedReal, "sub")), "~", "sub")
	})

	t.Run("unrelated path is not home", func(t *testing.T) {
		t.Setenv("HOME", "/home/env")
		p := homePowerline("/home/passwd")
		assert(t, cwdToPathSegments(p, "/etc/nginx"), "etc", "nginx")
	})
}

// Covers #253: -path-aliases-case-insensitive matches regardless of case.
func Test_pathAliases_caseInsensitive(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	newP := func(ci bool, mode string) *powerline {
		return &powerline{
			cwd:      "/home/test/Work/Projects",
			userInfo: user.User{HomeDir: "/home/test"},
			cfg: Config{
				CwdMode:                    mode,
				PathAliases:                AliasMap{"~/work/projects": "@P"},
				PathAliasesCaseInsensitive: ci,
			},
		}
	}

	t.Run("fancy: case-insensitive matches", func(t *testing.T) {
		got := pathSegmentPaths(cwdToPathSegments(newP(true, "fancy"), "/home/test/Work/Projects"))
		if len(got) != 1 || got[0] != "@P" {
			t.Fatalf("segments = %v, want [@P]", got)
		}
	})

	t.Run("fancy: case-sensitive does not match", func(t *testing.T) {
		got := pathSegmentPaths(cwdToPathSegments(newP(false, "fancy"), "/home/test/Work/Projects"))
		want := []string{"~", "Work", "Projects"}
		if len(got) != len(want) {
			t.Fatalf("segments = %v, want %v", got, want)
		}
	})

	t.Run("plain: case-insensitive matches", func(t *testing.T) {
		segs := segmentCwd(newP(true, "plain"))
		if len(segs) != 1 || segs[0].Content != "@P" {
			t.Fatalf("plain content = %v, want @P", segs)
		}
	})
}

// Covers #406: -path-aliases must also apply in -cwd-mode plain.
func Test_segmentCwd_plainPathAliases(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	p := &powerline{
		cwd:      "/home/test/work/projects/foo/bar",
		userInfo: user.User{HomeDir: "/home/test"},
		cfg: Config{
			CwdMode:     "plain",
			PathAliases: AliasMap{"~/work/projects/foo": "@FOO"},
		},
	}
	segs := segmentCwd(p)
	if len(segs) != 1 {
		t.Fatalf("plain cwd = %d segments, want 1", len(segs))
	}
	if segs[0].Content != "@FOO/bar" {
		t.Errorf("plain cwd content = %q, want %q", segs[0].Content, "@FOO/bar")
	}
}
