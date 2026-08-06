package auth

import (
	"errors"
	"time"
)

// Browser cookie import — the convenience half of D-002.
//
// LeetCode has no OAuth for third parties, so the session cookies a logged-in browser
// already holds are the only credential that exists. Rather than making the user dig
// them out of devtools, leetui reads them from the browser's own cookie store. leetgo
// takes the same approach (`credentials.from: browser`); the VS Code extension gets the
// same cookie by opening a webview and harvesting it from that session, which a
// terminal app has no equivalent of.
//
// WHERE THE COOKIES LIVE
//
// Chromium browsers (Chrome, Brave, Edge, Arc, Chromium) keep cookies in a SQLite
// database with encrypted values; see chromium.go for the schema and
// chromium_crypto.go for the key derivation. Firefox stores them in plaintext SQLite,
// handled in firefox.go.
//
// WHAT THIS DELIBERATELY DOES NOT DO
//
//   - It never writes a cookie anywhere except the OS keychain (see Store).
//   - It reads only leetcode.com cookies, and only the two names that are needed. The
//     rest of the cookie jar is never selected, never decrypted, never logged.
//   - It stays optional. Manual paste is the primary, portable path, and any failure
//     here falls back to it rather than blocking sign-in (D-002).

// ErrNoBrowserCookies means no supported browser held a usable LeetCode session.
var ErrNoBrowserCookies = errors.New("no leetcode session found in any browser")

// ImportTimeout bounds an import attempt. The keychain prompt is modal and the user may
// take a moment to answer it.
const ImportTimeout = 60 * time.Second

// Browser is a detected browser profile that can be imported from.
type Browser struct {
	// Name is what the user calls it, e.g. "Chrome" or "Brave".
	Name string
	// Profile is the profile folder name, e.g. "Default". Firefox uses a random one.
	Profile string
	// cookiePath is the cookie database.
	cookiePath string
	// keychainService/keychainAccount locate the decryption key. Empty for Firefox.
	keychainService string
	keychainAccount string
	// firefox marks the plaintext schema.
	firefox bool
}

// Label is what the picker shows.
func (b Browser) Label() string {
	if b.Profile != "" && b.Profile != "Default" {
		return b.Name + " · " + b.Profile
	}
	return b.Name
}
