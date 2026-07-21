package main

import (
	"os"
	"testing"
)

// Covers #290: the shlvl segment reads $SHLVL and renders only when the depth
// is at least MinShlvl.
func Test_segmentShLvl(t *testing.T) {
	p := &powerline{theme: Theme{JobsFg: 39, JobsBg: 238}, cfg: Config{MinShlvl: 2}}

	t.Run("hidden at base shell (SHLVL below min)", func(t *testing.T) {
		t.Setenv("SHLVL", "1")
		if segs := segmentShLvl(p); len(segs) != 0 {
			t.Fatalf("want no segment at SHLVL=1 with min 2, got %v", segs)
		}
	})

	t.Run("shown when nested (SHLVL >= min)", func(t *testing.T) {
		t.Setenv("SHLVL", "3")
		segs := segmentShLvl(p)
		if len(segs) != 1 || segs[0].Content != "3" {
			t.Fatalf("want a single segment with content 3, got %v", segs)
		}
		if segs[0].Foreground != 39 || segs[0].Background != 238 {
			t.Errorf("want the jobs palette 39/238, got %d/%d", segs[0].Foreground, segs[0].Background)
		}
	})

	t.Run("hidden when SHLVL is unset or invalid", func(t *testing.T) {
		os.Unsetenv("SHLVL")
		if segs := segmentShLvl(p); len(segs) != 0 {
			t.Fatalf("want no segment when SHLVL is unset, got %v", segs)
		}
		t.Setenv("SHLVL", "notanumber")
		if segs := segmentShLvl(p); len(segs) != 0 {
			t.Fatalf("want no segment when SHLVL is invalid, got %v", segs)
		}
	})
}
