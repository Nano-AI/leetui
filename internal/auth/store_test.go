package auth

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
)

// isolate points config.Dir at a temp directory and forces the keyring probe to report
// unavailable, so these tests exercise the fallback chain without touching the developer's
// real keychain or their real config.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.DirEnv, dir)
	t.Setenv(EnvSession, "")
	t.Setenv(EnvCSRF, "")

	// keyringAvailable caches behind a sync.Once, so tests must be able to pin it.
	old := keyringOK
	keyringOnce.Do(func() {})
	keyringOK = false
	t.Cleanup(func() { keyringOK = old })

	return dir
}

func sample() Credentials {
	return Credentials{Session: "eyJhbGciOiJIUzI1NiJ9.payload.sig", CSRF: "abc123DEF456", Username: "ada"}
}

func TestFileRoundTrip(t *testing.T) {
	dir := isolate(t)

	backend, err := Store(sample())
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if backend != BackendFile {
		t.Fatalf("stored to %q, want the file fallback", backend)
	}

	got, from, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if from != BackendFile {
		t.Errorf("loaded from %q, want %q", from, BackendFile)
	}
	if got.Session != sample().Session || got.CSRF != sample().CSRF {
		t.Errorf("round trip lost the cookies: %v", got)
	}
	if got.Username != "ada" {
		t.Errorf("username = %q, want it preserved", got.Username)
	}

	// The whole justification for writing a secret to disk is that only its owner can
	// read it. If the mode is wrong the fallback is not defensible.
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials file mode = %04o, want 0600", perm)
	}
}

// TestLooseFileIsRefused: a credentials file that others can read has already been
// exposed, and silently repairing it would hide that from the only person who can
// decide what to do about it.
func TestLooseFileIsRefused(t *testing.T) {
	dir := isolate(t)
	if _, err := Store(sample()); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, FileName)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Load()
	if err == nil {
		t.Fatal("a world-readable credentials file was accepted")
	}
	if !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("error %q does not tell the user how to fix it", err)
	}
}

// TestPartialFileIsNoCredentials: a file holding half a credential is the first-run
// case as far as the app is concerned, not a hard error that blocks a public board.
func TestPartialFileIsNoCredentials(t *testing.T) {
	dir := isolate(t)
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"session":"only-half"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Load(); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Load on a half-written file = %v, want ErrNoCredentials", err)
	}
}

func TestLoadWithNothingStored(t *testing.T) {
	isolate(t)
	if _, _, err := Load(); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Load on a clean machine = %v, want ErrNoCredentials", err)
	}
}

func TestClearRemovesTheFile(t *testing.T) {
	dir := isolate(t)
	if _, err := Store(sample()); err != nil {
		t.Fatal(err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("Clear left the credentials file behind")
	}
	if err := Clear(); err != nil {
		t.Errorf("Clear on an already-clear machine failed: %v", err)
	}
}

func TestStoreRefusesIncomplete(t *testing.T) {
	isolate(t)
	if _, err := Store(Credentials{Session: "only-one"}); err == nil {
		t.Error("stored a credential with no csrftoken")
	}
}

// TestEnvWins: an override that does not override is worse than no override, because
// the user has no way to see that it was ignored.
func TestEnvWins(t *testing.T) {
	isolate(t)
	if _, err := Store(sample()); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvSession, "env-session")
	t.Setenv(EnvCSRF, "env-csrf")

	got, from, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if from != BackendEnv {
		t.Errorf("loaded from %q, want %q", from, BackendEnv)
	}
	if got.Session != "env-session" {
		t.Errorf("session = %q, want the environment's", got.Session)
	}
}

// TestHalfSetEnvIsAnError: one variable without the other is a typo in a shell profile
// or a CI secret that failed to inject. Falling through would sign the user in as
// somebody they did not intend with nothing on screen to explain it.
func TestHalfSetEnvIsAnError(t *testing.T) {
	isolate(t)
	t.Setenv(EnvSession, "env-session")

	_, _, err := Load()
	if err == nil {
		t.Fatal("half-configured environment was silently ignored")
	}
	if !strings.Contains(err.Error(), EnvCSRF) {
		t.Errorf("error %q does not name the missing variable", err)
	}
}

func TestAvailableAlwaysOffersAStore(t *testing.T) {
	isolate(t)
	avail := Available()
	if len(avail) == 0 {
		t.Fatal("no backend available, so sign-in would be impossible")
	}
	if avail[len(avail)-1] != BackendFile {
		t.Errorf("last resort = %q, want the file backend", avail[len(avail)-1])
	}
	if Target() != avail[0] {
		t.Errorf("Target = %q, want the best available (%q)", Target(), avail[0])
	}
}

func TestBackendSecure(t *testing.T) {
	for _, tc := range []struct {
		b    Backend
		want bool
	}{
		{BackendKeyring, true},
		{BackendHelper, true},
		{BackendFile, false},
		{BackendEnv, false},
	} {
		if got := tc.b.Secure(); got != tc.want {
			t.Errorf("%q.Secure() = %v, want %v", tc.b, got, tc.want)
		}
	}
}
