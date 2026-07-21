package main

import (
	"os"
	"strconv"
	"testing"
)

// Covers #208: -colorize-hostname must honour PLGO_HOSTNAMEFG and
// PLGO_HOSTNAMEBG independently (previously both had to be set or neither
// applied), and an unmapped background must not fall back to colour 0 (black).
func Test_segmentHost_colorize(t *testing.T) {
	const host = "testhost"
	hashBg := getMd5(host)[0] % 128
	otherBg := uint8(42)
	if otherBg == hashBg {
		otherBg = 43
	}

	theme := Theme{
		HostnameFg: 250,
		HostnameColorizedFgMap: map[uint8]uint8{
			hashBg:  21,
			otherBg: 22,
		},
	}

	check := func(t *testing.T, wantFg, wantBg uint8) {
		t.Helper()
		p := &powerline{hostname: host, theme: theme, cfg: Config{ColorizeHostname: true}}
		segs := segmentHost(p)
		if len(segs) != 1 {
			t.Fatalf("segmentHost returned %d segments, want 1", len(segs))
		}
		if segs[0].Foreground != wantFg || segs[0].Background != wantBg {
			t.Errorf("fg/bg = %d/%d, want %d/%d",
				segs[0].Foreground, segs[0].Background, wantFg, wantBg)
		}
		if segs[0].Content != host {
			t.Errorf("content = %q, want %q", segs[0].Content, host)
		}
	}

	t.Run("no env: hash bg + mapped fg", func(t *testing.T) {
		os.Unsetenv("PLGO_HOSTNAMEFG")
		os.Unsetenv("PLGO_HOSTNAMEBG")
		check(t, 21, hashBg)
	})
	t.Run("only bg: fg tracks the chosen bg", func(t *testing.T) {
		os.Unsetenv("PLGO_HOSTNAMEFG")
		t.Setenv("PLGO_HOSTNAMEBG", strconv.Itoa(int(otherBg)))
		check(t, 22, otherBg)
	})
	t.Run("only fg: bg still from hash", func(t *testing.T) {
		os.Unsetenv("PLGO_HOSTNAMEBG")
		t.Setenv("PLGO_HOSTNAMEFG", "99")
		check(t, 99, hashBg)
	})
	t.Run("both env override", func(t *testing.T) {
		t.Setenv("PLGO_HOSTNAMEBG", strconv.Itoa(int(otherBg)))
		t.Setenv("PLGO_HOSTNAMEFG", "99")
		check(t, 99, otherBg)
	})
	t.Run("unmapped bg falls back to HostnameFg not black", func(t *testing.T) {
		os.Unsetenv("PLGO_HOSTNAMEFG")
		t.Setenv("PLGO_HOSTNAMEBG", "200")
		check(t, 250, 200)
	})
}
