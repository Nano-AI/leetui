package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHelper writes a shell script implementing the get/store/erase protocol against a
// file, points config.toml at it, and returns nothing but the side effects.
//
// A shell script is the right fixture here precisely because it is what a real user will
// write. The documented pass(1) integration is four lines of exactly this shape, so the
// test exercises the same seam the docs promise.
func fakeHelper(t *testing.T, dir string, body string) {
	t.Helper()

	bin := filepath.Join(dir, "leetui-credential-fake")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// LookPath resolves through PATH, so the helper has to be reachable the same way a
	// real one would be.
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfg, []byte("[auth]\nhelper = \"leetui-credential-fake\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHelperRoundTrip(t *testing.T) {
	dir := isolate(t)
	store := filepath.Join(dir, "vault.json")

	fakeHelper(t, dir, `
case "$1" in
  store) cat > "`+store+`" ;;
  get)   cat "`+store+`" 2>/dev/null || exit 1 ;;
  erase) rm -f "`+store+`" ;;
esac
`)

	backend, err := Store(sample())
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if backend != BackendHelper {
		t.Fatalf("stored to %q, want the helper", backend)
	}

	// The helper is preferred over the file, so nothing should have reached disk here.
	if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
		t.Error("a configured helper still left a plaintext credentials file behind")
	}

	got, from, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if from != BackendHelper {
		t.Errorf("loaded from %q, want the helper", from)
	}
	if got.Session != sample().Session || got.CSRF != sample().CSRF {
		t.Errorf("helper round trip lost the cookies: %v", got)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, _, err := Load(); err == nil {
		t.Error("Clear did not reach the helper")
	}
}

// TestHelperWithNothingStoredFallsThrough: a helper exits non-zero when it has no entry,
// which is the ordinary first-run case rather than a fault. Treating it as an error
// would make a freshly configured helper look broken.
func TestHelperWithNothingStoredFallsThrough(t *testing.T) {
	dir := isolate(t)
	fakeHelper(t, dir, `exit 1`)

	_, _, err := Load()
	if err == nil {
		t.Fatal("expected ErrNoCredentials on a clean machine")
	}
	if !strings.Contains(err.Error(), "no stored credentials") {
		t.Errorf("an empty helper surfaced as %v, want ErrNoCredentials", err)
	}
}

// TestBrokenHelperIsReported: a helper that is configured but missing has to say so.
// Silently writing the session to a file instead would be the exact failure this whole
// design exists to avoid.
func TestBrokenHelperIsReported(t *testing.T) {
	dir := isolate(t)
	cfg := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfg, []byte("[auth]\nhelper = \"leetui-helper-that-does-not-exist\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	backend, err := Store(sample())
	if err == nil {
		t.Fatalf("a missing helper silently fell through to %q", backend)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not say the helper is missing", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, FileName)); statErr == nil {
		t.Error("a broken helper caused the session to be written to a file anyway")
	}
}

// TestHelperStderrReachesTheUser: stdin and stdout carry the secret and are never
// quoted, but stderr is where a helper's author puts the message for this moment.
func TestHelperStderrReachesTheUser(t *testing.T) {
	dir := isolate(t)
	fakeHelper(t, dir, `echo "gpg: decryption failed" >&2; exit 2`)

	_, err := Store(sample())
	if err == nil {
		t.Fatal("a failing helper reported success")
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("error %q dropped the helper's own explanation", err)
	}
}
