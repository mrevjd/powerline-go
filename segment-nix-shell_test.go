package main

import (
	"os"
	"testing"
)

// Covers #375: the nix-shell segment appends $NIX_SHELL_PACKAGES when set.
func Test_segmentNixShell(t *testing.T) {
	p := &powerline{symbols: SymbolTemplate{NixShellIndicator: "NIX"}}

	t.Run("not in a nix shell yields no segment", func(t *testing.T) {
		os.Unsetenv("IN_NIX_SHELL")
		os.Unsetenv("NIX_SHELL_PACKAGES")
		if segs := segmentNixShell(p); len(segs) != 0 {
			t.Fatalf("want no segment outside a nix shell, got %v", segs)
		}
	})

	t.Run("in a nix shell without packages shows only the indicator", func(t *testing.T) {
		t.Setenv("IN_NIX_SHELL", "impure")
		os.Unsetenv("NIX_SHELL_PACKAGES")
		segs := segmentNixShell(p)
		if len(segs) != 1 || segs[0].Content != "NIX" {
			t.Fatalf("want a single segment 'NIX', got %v", segs)
		}
	})

	t.Run("in a nix shell with packages appends them", func(t *testing.T) {
		t.Setenv("IN_NIX_SHELL", "impure")
		t.Setenv("NIX_SHELL_PACKAGES", "go gopls")
		segs := segmentNixShell(p)
		if len(segs) != 1 || segs[0].Content != "NIX go gopls" {
			t.Fatalf("want 'NIX go gopls', got %v", segs)
		}
	})
}
