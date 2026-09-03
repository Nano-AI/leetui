// Package auth acquires and stores LeetCode session credentials.
//
// LeetCode has no public API and no third-party OAuth, so everything rides on the two
// cookies a logged-in browser session holds (D-002):
//
//	LEETCODE_SESSION  the session JWT
//	csrftoken         required as the x-csrftoken header on every mutating request
//
// STORAGE. Credentials go to the best store the machine actually has, in this order:
//
//	keychain  the OS keyring - preferred, and the only tier that needs no setup
//	helper    a user-configured command (config.toml [auth] helper), Docker's model
//	file      credentials.json in the config dir, mode 0600 - the last resort
//
// The file tier exists because the alternative is that leetui simply cannot sign in on
// a Linux box with no Secret Service, which is not a defensible answer. What is not
// negotiable is that the fallback is never silent: see Backend, which every caller of
// Store and Load receives so it can tell the user where the secret went.
//
// We deliberately do NOT encrypt the file with a key kept beside it. That raises no
// attacker's cost (anyone who can read the ciphertext can read the key) while letting
// the tool claim a security property it does not have. Docker reached the same
// conclusion and answered it with external helpers rather than self-encryption, which
// is why the helper tier exists.
//
// SECURITY INVARIANTS — these are not style preferences:
//
//   - Credentials are never logged, never included in an error message, and never
//     written to a crash dump. Redact() exists for anything that must be displayed.
//   - String returns a redacted form so an accidental %v or %s cannot leak a session.
//   - A credentials file that is readable by anyone but its owner is refused, not
//     repaired: a permission that loosened is a fact the user needs to hear.
package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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

// Sniff reports whether a string contains BOTH cookies by name.
//
// It is the smart-paste hook for the two-field sign-in form: when what landed in either
// field turns out to be a whole cookie header or a cURL command, both fields can be
// filled from that one paste instead of making the user split it by hand.
//
// Deliberately all-or-nothing. Half a credential distributed across the form is worse
// than leaving the paste alone, because the user cannot see which half moved.
func Sniff(s string) (Credentials, bool) {
	c, err := Parse(s)
	return c, err == nil
}

// Clean reduces one pasted field to the bare cookie value.
//
// The user is copying out of a devtools table, a password manager, or a shell variable,
// and each one decorates the value differently. Rather than reject the decoration,
// which is what the old single-field form did, with "found neither LEETCODE_SESSION nor
// csrftoken" as the only explanation - strip it:
//
//	LEETCODE_SESSION=eyJ…    a name= prefix, in either = or : form
//	"eyJ…"                   surrounding quotes, from JSON or a shell
//	eyJ…;                    a trailing semicolon, from a cookie header
//
// Whitespace goes too, including the newline a terminal appends to a bracketed paste.
func Clean(raw string) string {
	v := strings.TrimSpace(raw)

	// A name= prefix, only when it names a cookie we actually want. Stripping any
	// leading "word=" would silently mangle a value that legitimately contains one.
	for _, name := range []string{"LEETCODE_SESSION", "csrftoken"} {
		if len(v) > len(name) && strings.EqualFold(v[:len(name)], name) {
			if rest := strings.TrimLeft(v[len(name):], " \t"); strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, ":") {
				v = strings.TrimSpace(rest[1:])
				break
			}
		}
	}

	// Quotes, but only as a matched pair: a lone quote is part of the value, or a
	// sign the paste is truncated, and either way is not ours to remove.
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}

	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(v), ";"))
}

// ---------------------------------------------------------------------------
// Onboarding copy
// ---------------------------------------------------------------------------

// PasteSteps is the manual route, as short directions rather than an explanation.
// The user is mid-task and wants the steps, not a description of how the app works.
//
// Each line is kept under 52 columns so it fits the sign-in panel without wrapping.
func PasteSteps() []string {
	return []string{
		"On leetcode.com: devtools → Application → Cookies",
		"Copy each value into the matching field above.",
		"A whole cookie header or cURL command works too:",
		"paste it into either field and both fill in.",
	}
}
