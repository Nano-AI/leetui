package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogAppendsAndTimestamps(t *testing.T) {
	t.Setenv(DirEnv, t.TempDir())
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	l, path, err := OpenLog()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	l.Printf("first %d", 1)
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Appends rather than truncates: the interesting run is often the one BEFORE the
	// one you are looking at, and losing it to a restart means reproducing twice.
	l2, _, err := OpenLog()
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	l2.Printf("second")
	l2.Close()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(body)
	for _, want := range []string{"first 1", "second"} {
		if !strings.Contains(out, want) {
			t.Errorf("log is missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, ":") {
		t.Error("lines carry no timestamp")
	}
	if filepath.Base(path) != "debug.log" {
		t.Errorf("log went to %q", path)
	}
}

// A discard Logger must be safe to call, so no call site has to branch on whether
// tracing is on.
func TestDiscardIsSafe(t *testing.T) {
	d := Discard()
	d.Printf("nothing %s", "happens")
	if err := d.Close(); err != nil {
		t.Errorf("close: %v", err)
	}

	var nilLog *Logger
	nilLog.Printf("also fine")
	if err := nilLog.Close(); err != nil {
		t.Errorf("nil close: %v", err)
	}
}

func TestDebugRequested(t *testing.T) {
	t.Setenv(DebugEnv, "")
	if DebugRequested() {
		t.Error("empty env turned tracing on")
	}
	t.Setenv(DebugEnv, "1")
	if !DebugRequested() {
		t.Error("env did not turn tracing on")
	}
}
