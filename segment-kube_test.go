package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Covers #419: readKubeConfig must skip non-regular files (e.g. a directory,
// which is what an empty KUBECONFIG entry resolves to) instead of reading them,
// which can panic on FUSE/Ceph filesystems.
func Test_readKubeConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("directory is skipped with an error", func(t *testing.T) {
		if err := readKubeConfig(&KubeConfig{}, dir); err == nil {
			t.Fatal("expected an error reading a directory, got nil")
		}
	})

	t.Run("missing file returns an error", func(t *testing.T) {
		if err := readKubeConfig(&KubeConfig{}, filepath.Join(dir, "nope")); err == nil {
			t.Fatal("expected an error for a missing file, got nil")
		}
	})

	t.Run("valid config parses", func(t *testing.T) {
		f := filepath.Join(dir, "config")
		content := "current-context: ctx\ncontexts:\n- name: ctx\n  context:\n    namespace: ns\n"
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := &KubeConfig{}
		if err := readKubeConfig(cfg, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CurrentContext != "ctx" {
			t.Errorf("CurrentContext = %q, want ctx", cfg.CurrentContext)
		}
	})
}
