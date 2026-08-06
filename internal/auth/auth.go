// Package auth acquires and stores LeetCode session credentials.
//
// LeetCode has no public API and no third-party OAuth, so everything rides on the two
// cookies a logged-in browser session holds (D-002):
//
//	LEETCODE_SESSION  the session JWT
//	csrftoken         required as the x-csrftoken header on every mutating request
//
// SECURITY INVARIANTS — these are not style preferences:
//
//   - Credentials live in the OS keychain. Never in config.toml, never in the workspace,
//     never in a dotfile.
//   - Credentials are never logged, never included in an error message, and never
//     written to a crash dump. Redact() exists for anything that must be displayed.
//   - String returns a redacted form so an accidental %v or %s cannot leak a session.
package auth

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/zalando/go-keyring"
)

// keyring identifiers.
const (
	service     = "leetui"
	keySession  = "leetcode_session"
	keyCSRF     = "leetcode_csrftoken"
	keyUsername = "leetcode_username" // cosmetic only; shown in the rail
)

// ErrNoCredentials means nothing is stored yet — the user has not authenticated.
var ErrNoCredentials = errors.New("no stored credentials")

// ErrSessionExpired means LeetCode rejected the stored session. The TUI surfaces this
// as an in-place re-auth prompt rather than failing the user's action outright.
var ErrSessionExpired = errors.New("leetcode session expired")

// Credentials is a LeetCode browser session.
type Credentials struct {
	Session  string
	CSRF     string
	Username string // cosmetic; may be empty
}

// Valid reports whether both required cookies are present.
func (c Credentials) Valid() bool {
	return c.Session != "" && c.CSRF != ""
}

// Cookie returns the Cookie header value for an authenticated request.
//
// Never log the result.
func (c Credentials) Cookie() string {
	return fmt.Sprintf("LEETCODE_SESSION=%s; csrftoken=%s", c.Session, c.CSRF)
}

// String is deliberately redacted so an accidental fmt verb cannot leak a session.
func (c Credentials) String() string {
	return fmt.Sprintf("Credentials{session:%s csrf:%s user:%q}",
		Redact(c.Session), Redact(c.CSRF), c.Username)
}

// Redact reduces a secret to a length-tagged fingerprint that is safe to display.
// It shows enough to tell two sessions apart and not enough to use one.
func Redact(secret string) string {
	if secret == "" {
		return "<empty>"
	}
	if len(secret) <= 8 {
		return "<redacted>"
	}
	return fmt.Sprintf("%s…%s(%d)", secret[:4], secret[len(secret)-4:], len(secret))
}

// ---------------------------------------------------------------------------
// Parsing pasted input
// ---------------------------------------------------------------------------

var (
	reSession = regexp.MustCompile(`LEETCODE_SESSION["' :=]+([A-Za-z0-9._\-]+)`)
	reCSRF    = regexp.MustCompile(`csrftoken["' :=]+([A-Za-z0-9._\-]+)`)
)

// Parse extracts credentials from whatever the user pasted.
//
// It deliberately accepts several shapes, because the user is copying out of a browser
// and the exact form depends on where they copied from:
//
//   - a raw Cookie header: "LEETCODE_SESSION=eyJ...; csrftoken=abc123"
//   - devtools' "Copy as cURL" output, cookies embedded among other headers
//   - a cookie-exporter JSON blob containing both names
//   - the two values on separate lines
//
// Rather than demand one format, it scans for the two names anywhere in the input.
func Parse(pasted string) (Credentials, error) {
	var c Credentials

	if m := reSession.FindStringSubmatch(pasted); len(m) == 2 {
		c.Session = m[1]
	}
	if m := reCSRF.FindStringSubmatch(pasted); len(m) == 2 {
		c.CSRF = m[1]
	}

	switch {
	case c.Session == "" && c.CSRF == "":
		return c, errors.New("found neither LEETCODE_SESSION nor csrftoken in the pasted text")
	case c.Session == "":
		return c, errors.New("found csrftoken but not LEETCODE_SESSION")
	case c.CSRF == "":
		return c, errors.New("found LEETCODE_SESSION but not csrftoken")
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// Keychain storage
// ---------------------------------------------------------------------------

// Store writes credentials to the OS keychain.
func Store(c Credentials) error {
	if !c.Valid() {
		return errors.New("refusing to store incomplete credentials")
	}
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

// Load reads credentials from the OS keychain.
//
// Returns ErrNoCredentials when nothing is stored, which is the ordinary first-run case
// and not a failure.
func Load() (Credentials, error) {
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

// Clear removes stored credentials. Missing entries are not an error, so Clear is safe
// to call as part of sign-out regardless of current state.
func Clear() error {
	var errs []error
	for _, k := range []string{keySession, keyCSRF, keyUsername} {
		if err := keyring.Delete(service, k); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// Onboarding copy
// ---------------------------------------------------------------------------

// PasteSteps is the manual fallback, as short directions rather than an explanation.
// The user is mid-task and wants the steps, not a description of how the app works.
//
// Each line is kept under 52 columns so it fits the sign-in panel without wrapping.
func PasteSteps() []string {
	return []string{
		"On leetcode.com: devtools → Application → Cookies",
		"Copy LEETCODE_SESSION and csrftoken",
		"Paste here in any form — cookie header, cURL, or",
		"the two values on separate lines.",
	}
}
