package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeGCloudConfig(t *testing.T, home, name, contents string) string {
	t.Helper()
	dir := filepath.Join(home, ".config", "gcloud", "configurations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config_"+name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, ".config", "gcloud")
}

// gcloud only writes active_config once a second configuration exists, so its
// absence means "default", not a reason to shell out to gcloud.
func TestGCPProjectFromFileWithoutActiveConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeGCloudConfig(t, home, "default", "[core]\nproject = my-project\n")

	got, err := getGCPProjectFromFile()
	if err != nil {
		t.Fatalf("getGCPProjectFromFile: %v", err)
	}
	if got != "my-project" {
		t.Errorf("project = %q, want %q", got, "my-project")
	}
}

// A section before [core] used to have its offset applied twice to the
// remaining bytes, which skipped lines or sliced past the end and panicked.
func TestGCPProjectFromFileCoreNotFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gcloud := writeGCloudConfig(t, home, "work", "[auth]\naccount = someone\n[core]\nproject = work-project\n[compute]\nzone = z\n")
	if err := os.WriteFile(filepath.Join(gcloud, "active_config"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := getGCPProjectFromFile()
	if err != nil {
		t.Fatalf("getGCPProjectFromFile: %v", err)
	}
	if got != "work-project" {
		t.Errorf("project = %q, want %q", got, "work-project")
	}
}
