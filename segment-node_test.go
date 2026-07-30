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
	t.Chdir(dir)

	p := &powerline{symbols: SymbolTemplate{NodeIndicator: "N"}}

	// Assert the project detection directly as well as the rendered segments: an
	// environment without node installed produces no segments either way, so the
	// segment count alone would pass even if detection regressed.
	t.Run("no package.json yields no segments", func(t *testing.T) {
		if _, inNodeProject := readPackageJSON(); inNodeProject {
			t.Error("expected no node project without a package.json")
		}
		if segs := segmentNode(p); len(segs) != 0 {
			t.Fatalf("expected no segments outside a node project, got %d: %v", len(segs), segs)
		}
	})

	t.Run("a package.json directory is not a node project", func(t *testing.T) {
		pkg := filepath.Join(dir, "package.json")
		if err := os.Mkdir(pkg, 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(pkg)

		if _, inNodeProject := readPackageJSON(); inNodeProject {
			t.Error("expected a package.json directory not to count as a node project")
		}
		if segs := segmentNode(p); len(segs) != 0 {
			t.Fatalf("expected no segments when package.json is a directory, got %d: %v", len(segs), segs)
		}
	})

	// An unreadable or malformed package.json still means "this is a node project",
	// so the node segment keeps rendering and only the version is dropped. That
	// distinction is why readPackageJSON returns a bool rather than just "".
	t.Run("malformed package.json is still a node project with no version", func(t *testing.T) {
		pkg := filepath.Join(dir, "package.json")
		if err := os.WriteFile(pkg, []byte(`{"version":`), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(pkg)

		version, inNodeProject := readPackageJSON()
		if !inNodeProject {
			t.Error("a malformed package.json should still count as a node project")
		}
		if version != "" {
			t.Errorf("version = %q, want empty for malformed JSON", version)
		}
		for _, s := range segmentNode(p) {
			if s.Name == "node-segment" {
				t.Errorf("expected no package-version segment, got %q", s.Content)
			}
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
