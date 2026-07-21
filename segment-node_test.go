package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers #356: the node segment must only appear inside a node project
// (package.json present), not whenever `node` is merely on PATH.
func Test_segmentNode(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	p := &powerline{symbols: SymbolTemplate{NodeIndicator: "N"}}

	t.Run("no package.json yields no segments", func(t *testing.T) {
		if segs := segmentNode(p); len(segs) != 0 {
			t.Fatalf("expected no segments outside a node project, got %d: %v", len(segs), segs)
		}
	})

	t.Run("package.json present surfaces the package version", func(t *testing.T) {
		pkg := filepath.Join(dir, "package.json")
		if err := os.WriteFile(pkg, []byte(`{"version":"1.2.3"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(pkg)

		found := false
		for _, s := range segmentNode(p) {
			if strings.Contains(s.Content, "1.2.3") {
				found = true
			}
		}
		if !found {
			t.Error("expected a segment containing the package version 1.2.3")
		}
	})
}
