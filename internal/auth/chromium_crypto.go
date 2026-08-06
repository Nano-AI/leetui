package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"errors"
	"fmt"
	"runtime"

	"github.com/zalando/go-keyring"
)

// chromiumKey derives the AES key for a Chromium profile.
//
// On macOS this reads the login keychain, which prompts the user for permission. That
// prompt is a feature: granting a terminal app access to browser cookies should be an
// explicit, visible decision.
func chromiumKey(b Browser) ([]byte, error) {
	const (
		salt   = "saltysalt"
		keyLen = 16
	)

	switch runtime.GOOS {
	case "darwin":
		password, err := keyring.Get(b.keychainService, b.keychainAccount)
		if err != nil {
			return nil, fmt.Errorf("read %q from the keychain: %w", b.keychainService, err)
		}
		return pbkdf2.Key(sha1.New, password, []byte(salt), 1003, keyLen)

	case "linux":
		// The desktop secret service may hold a per-browser password; unlocked profiles
		// fall back to a fixed literal that Chromium uses when no keyring is available.
		password := "peanuts"
		if p, err := keyring.Get(b.keychainService, b.keychainAccount); err == nil && p != "" {
			password = p
		}
		return pbkdf2.Key(sha1.New, password, []byte(salt), 1, keyLen)

	default:
		return nil, fmt.Errorf("browser import is not supported on %s — paste the cookies instead", runtime.GOOS)
	}
}

// decryptChromium reverses Chromium's cookie encryption.
func decryptChromium(enc, key []byte) (string, error) {
	if len(enc) < 3 {
		return "", errors.New("ciphertext too short")
	}

	version := string(enc[:3])
	switch version {
	case "v10", "v11":
	default:
		// Windows "v20" values are wrapped with DPAPI and need a different path.
		return "", fmt.Errorf("unsupported cookie encryption %q", version)
	}
	body := enc[3:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("build cipher: %w", err)
	}
	if len(body)%aes.BlockSize != 0 || len(body) == 0 {
		return "", errors.New("ciphertext is not a whole number of blocks")
	}

	// Chromium uses a fixed IV of 16 spaces.
	iv := bytes.Repeat([]byte{' '}, aes.BlockSize)
	out := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, body)

	out, err = stripPKCS7(out)
	if err != nil {
		return "", err
	}
	return string(stripDomainHash(out)), nil
}

// stripPKCS7 removes the block padding, rejecting malformed input rather than returning
// a silently truncated cookie.
func stripPKCS7(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("empty plaintext")
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(b) {
		return nil, errors.New("bad padding")
	}
	for _, c := range b[len(b)-pad:] {
		if int(c) != pad {
			return nil, errors.New("bad padding")
		}
	}
	return b[:len(b)-pad], nil
}

// stripDomainHash removes the 32-byte SHA-256 of the cookie's domain that Chrome began
// prepending to plaintext in late 2024.
//
// There is no version marker for it, so it is detected by content: a cookie value is
// printable ASCII, and a hash almost never is.
func stripDomainHash(b []byte) []byte {
	const hashLen = 32
	if len(b) <= hashLen {
		return b
	}
	if isPrintableASCII(b[:hashLen]) {
		return b // no hash prefix; the value starts here
	}
	return b[hashLen:]
}

func isPrintableASCII(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}
