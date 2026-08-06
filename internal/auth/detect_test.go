package auth

import "testing"

// TestDetectBrowsers lists browser profiles found on this machine. It deliberately does
// NOT import: reading the decryption key opens a modal keychain prompt, which cannot be
// answered by an unattended test.
func TestDetectBrowsers(t *testing.T) {
	bs := DetectBrowsers()
	for _, b := range bs {
		t.Logf("%-16s profile=%-12s firefox=%v", b.Label(), b.Profile, b.firefox)
	}
	if len(bs) == 0 {
		t.Skip("no supported browser on this machine")
	}
	for _, b := range bs {
		if b.cookiePath == "" {
			t.Errorf("%s has no cookie path", b.Label())
		}
		if !b.firefox && b.keychainService == "" {
			t.Errorf("%s is chromium but has no keychain service", b.Label())
		}
	}
}

func TestStripDomainHash(t *testing.T) {
	plain := []byte("abc123sessiontoken")
	if got := string(stripDomainHash(plain)); got != string(plain) {
		t.Errorf("plain value was truncated: %q", got)
	}

	// A 32-byte non-printable prefix is a domain hash and must be removed.
	hashed := append(make([]byte, 32), []byte("sessiontoken")...)
	if got := string(stripDomainHash(hashed)); got != "sessiontoken" {
		t.Errorf("hash prefix not stripped: %q", got)
	}
}

func TestStripPKCS7(t *testing.T) {
	if _, err := stripPKCS7([]byte{}); err == nil {
		t.Error("empty plaintext accepted")
	}
	if _, err := stripPKCS7([]byte{1, 2, 99}); err == nil {
		t.Error("bad padding accepted")
	}
	out, err := stripPKCS7([]byte{'h', 'i', 2, 2})
	if err != nil || string(out) != "hi" {
		t.Errorf("stripPKCS7 = %q, %v", out, err)
	}
}
