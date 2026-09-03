package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

// Backend names where a credential lives.
//
// Every Store and Load returns one. That is the whole point: the failure mode this
// package is built to avoid is the one the GitHub CLI is repeatedly asked to fix, where
// a keyring that is not there silently downgrades to a file and nothing ever says so.
// A caller that has a Backend in hand has no excuse for not telling the user.
type Backend string

const (
	BackendKeyring Backend = "keychain"
	BackendHelper  Backend = "helper"
	BackendFile    Backend = "file"
	BackendEnv     Backend = "environment"
	BackendNone    Backend = ""
)

// Secure reports whether this backend keeps the secret out of readable plaintext.
//
// The file backend is the only one that does not, and the sign-in panel and doctor both
// need to say so in their own words.
func (b Backend) Secure() bool { return b == BackendKeyring || b == BackendHelper }

// Describe is one clause fit to drop into a sentence the user reads.
func (b Backend) Describe() string {
	switch b {
	case BackendKeyring:
		return "your OS keychain"
	case BackendHelper:
		return "your credential helper"
	case BackendFile:
		return "a 0600 file"
	case BackendEnv:
		return "the environment"
	default:
		return "nowhere"
	}
}

// keyring identifiers.
const (
	service     = "leetui"
	keySession  = "leetcode_session"
	keyCSRF     = "leetcode_csrftoken"
	keyUsername = "leetcode_username"
)

// ---------------------------------------------------------------------------
// The chain
// ---------------------------------------------------------------------------

// Store writes credentials to the best available backend and reports which one took them.
//
// The tiers are tried in descending order of safety, but a tier is skipped ONLY when it
// is genuinely unavailable. A keychain that exists and refuses the write is an error the
// user needs to see, not a reason to quietly write their session to a file instead.
func Store(c Credentials) (Backend, error) {
	if !c.Valid() {
		return BackendNone, errors.New("refusing to store incomplete credentials")
	}

	if keyringAvailable() {
		if err := storeKeyring(c); err != nil {
			return BackendNone, err
		}
		return BackendKeyring, nil
	}

	if h := helperCommand(); h != "" {
		if err := storeHelper(h, c); err != nil {
			return BackendNone, err
		}
		return BackendHelper, nil
	}

	if err := storeFile(c); err != nil {
		return BackendNone, err
	}
	return BackendFile, nil
}

// Load reads credentials from the first backend that has them.
//
// The environment goes first so a CI job, a `pass`-driven shell, or a second profile can
// override whatever is on the machine without having to clear it. After that the order
// matches Store, so the tier that wrote is the tier that reads.
//
// Returns ErrNoCredentials when nothing is stored anywhere, which is the ordinary
// first-run case and not a failure.
func Load() (Credentials, Backend, error) {
	if c, ok, err := loadEnv(); err != nil {
		return Credentials{}, BackendNone, err
	} else if ok {
		return c, BackendEnv, nil
	}

	if keyringAvailable() {
		c, err := loadKeyring()
		if err == nil {
			return c, BackendKeyring, nil
		}
		if !errors.Is(err, ErrNoCredentials) {
			return Credentials{}, BackendNone, err
		}
	}

	if h := helperCommand(); h != "" {
		c, err := loadHelper(h)
		if err == nil {
			return c, BackendHelper, nil
		}
		if !errors.Is(err, ErrNoCredentials) {
			return Credentials{}, BackendNone, err
		}
	}

	c, err := loadFile()
	if err == nil {
		return c, BackendFile, nil
	}
	return Credentials{}, BackendNone, err
}

// Clear removes stored credentials from every backend.
//
// Every tier is swept regardless of which one Load would have picked, because a machine
// that gained a keychain since sign-in would otherwise leave the old file behind: still
// readable, and no longer visible to the user through any part of the app.
//
// Missing entries are not an error, so Clear is safe to call regardless of current state.
func Clear() error {
	var errs []error
	if keyringAvailable() {
		errs = append(errs, clearKeyring())
	}
	if h := helperCommand(); h != "" {
		errs = append(errs, clearHelper(h))
	}
	errs = append(errs, clearFile())
	return errors.Join(errs...)
}

// Available lists the backends this machine can actually use, best first.
//
// doctor prints it, and the sign-in panel uses the head of it to tell the user where
// their cookies are about to go before they commit to typing them.
func Available() []Backend {
	var out []Backend
	if keyringAvailable() {
		out = append(out, BackendKeyring)
	}
	if helperCommand() != "" {
		out = append(out, BackendHelper)
	}
	// The file backend is always available: the config dir is ours to create.
	return append(out, BackendFile)
}

// Target is the backend a Store would land on right now, for disclosure before the fact.
func Target() Backend { return Available()[0] }

// ---------------------------------------------------------------------------
// Keyring backend
// ---------------------------------------------------------------------------

var (
	keyringOnce sync.Once
	keyringOK   bool
)

// keyringAvailable reports whether an OS keyring is actually reachable.
//
// This is a question go-keyring does not answer directly, so we ask by reading and
// classifying the failure. A miss (ErrNotFound) means the service answered, which is
// exactly what we want to know. A transport failure means there is nothing listening:
// on Linux that is a machine with no Secret Service on the session bus, which is the
// ordinary case on a headless box, over SSH, or under a window manager that ships no
// keyring agent.
//
// Probed once and cached. A keyring does not appear partway through a session, and
// re-probing would put a D-Bus round trip on the path of every credential read.
func keyringAvailable() bool {
	keyringOnce.Do(func() {
		_, err := keyring.Get(service, keySession)
		keyringOK = err == nil || errors.Is(err, keyring.ErrNotFound)
	})
	return keyringOK
}

func storeKeyring(c Credentials) error {
	if err := keyring.Set(service, keySession, c.Session); err != nil {
		return fmt.Errorf("store session in keychain: %w", err)
	}
	if err := keyring.Set(service, keyCSRF, c.CSRF); err != nil {
		return fmt.Errorf("store csrf token in keychain: %w", err)
	}
	if c.Username != "" {
		// Cosmetic; a failure here must not fail authentication.
		_ = keyring.Set(service, keyUsername, c.Username)
	}
	return nil
}

func loadKeyring() (Credentials, error) {
	var c Credentials

	session, err := keyring.Get(service, keySession)
	if errors.Is(err, keyring.ErrNotFound) {
		return c, ErrNoCredentials
	}
	if err != nil {
		return c, fmt.Errorf("read session from keychain: %w", err)
	}

	csrf, err := keyring.Get(service, keyCSRF)
	if errors.Is(err, keyring.ErrNotFound) {
		return c, ErrNoCredentials
	}
	if err != nil {
		return c, fmt.Errorf("read csrf token from keychain: %w", err)
	}

	c.Session, c.CSRF = session, csrf
	if u, err := keyring.Get(service, keyUsername); err == nil {
		c.Username = u
	}
	return c, nil
}

func clearKeyring() error {
	var errs []error
	for _, k := range []string{keySession, keyCSRF, keyUsername} {
		if err := keyring.Delete(service, k); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// Shared shape
// ---------------------------------------------------------------------------

// stored is the on-the-wire form for both the file and the helper.
//
// Field names match the cookies rather than the Go fields, so a helper script written
// against Docker's protocol reads the way its author expects.
type stored struct {
	Session  string `json:"session"`
	CSRF     string `json:"csrftoken"`
	Username string `json:"username,omitempty"`
}

func (s stored) creds() Credentials {
	return Credentials{
		Session:  strings.TrimSpace(s.Session),
		CSRF:     strings.TrimSpace(s.CSRF),
		Username: strings.TrimSpace(s.Username),
	}
}

func toStored(c Credentials) stored {
	return stored{Session: c.Session, CSRF: c.CSRF, Username: c.Username}
}
