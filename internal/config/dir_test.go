package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDirRespectsTheOverride is the guard for a bug that reached a live install: the test
// suite called Save(), which resolved to the developer's REAL config.toml and wrote a
// t.TempDir() workspace into it, blanking their editor on the way past.
func TestDirRespectsTheOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "profile")
	t.Setenv(DirEnv, want)

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("Dir did not create the directory: %v", err)
	}
}

// TestSaveStaysInsideTheOverride: the whole point is that a Save during a test cannot
// escape the sandbox it was given.
func TestSaveStaysInsideTheOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DirEnv, dir)

	cfg := Default()
	cfg.Workspace = "/tmp/somewhere-only-this-test-knows"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err != nil {
		t.Fatalf("Save did not write inside the override: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Workspace != cfg.Workspace {
		t.Errorf("round trip lost the workspace: %q", loaded.Workspace)
	}
}

// TestEmptyOverrideIsIgnored: an unset or blank variable must not send config to "".
func TestEmptyOverrideIsIgnored(t *testing.T) {
	t.Setenv(DirEnv, "   ")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got == "" || got == "   " {
		t.Errorf("a blank override was honoured: %q", got)
	}
}
