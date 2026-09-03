package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Nano-AI/leetui/internal/config"
)

// A credential helper is an external command that holds the secret for us.
//
// This is the tier that makes the plaintext fallback defensible. It is the only way to
// get real encryption on a machine with no keychain, because the key ends up somewhere
// other than beside the ciphertext: a GPG agent, a passphrase prompt, a TPM. Encrypting
// the file ourselves with a key stored next to it would look like security without
// being any, which is the reasoning Docker followed to credential helpers and away from
// base64-in-config.json.
//
// The protocol is Docker's, because it is small and its shape is already familiar:
//
//	<helper> get      stdout: {"session":"…","csrftoken":"…","username":"…"}
//	<helper> store    stdin:  that same JSON
//	<helper> erase
//
// A four-line pass(1) wrapper is a complete implementation; see docs/AGENTS.md.
//
// Exit status is the contract. A non-zero `get` means "nothing stored", not "broken":
// that is what lets a first run fall through to the next tier instead of erroring on a
// helper that simply has no entry yet.

// helperTimeout bounds a helper run. It matches ImportTimeout because the reason is the
// same: a helper may prompt for a passphrase or wait on a hardware token, and the user
// needs time to answer without leetui deciding the helper is hung.
const helperTimeout = ImportTimeout

// helperCommand returns the configured helper, or "" when none is set.
//
// Config failures resolve to "" rather than an error. A helper is opt-in, and a
// malformed config.toml already reports itself through config.Load at startup; making
// every credential read fail a second time for it helps nobody.
func helperCommand() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Auth.Helper)
}

// runHelper executes one helper verb.
//
// stdin and stdout both carry the session, so neither is ever logged or folded into an
// error. Only stderr is quoted back, and only on failure, because that is where a helper
// puts the message its author wrote for this moment.
func runHelper(cmd, verb string, in []byte) ([]byte, error) {
	bin, err := exec.LookPath(cmd)
	if err != nil {
		return nil, fmt.Errorf("credential helper %q not found: %w", cmd, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
	defer cancel()

	var out, errBuf bytes.Buffer
	c := exec.CommandContext(ctx, bin, verb)
	c.Stdout = &out
	c.Stderr = &errBuf
	if in != nil {
		c.Stdin = bytes.NewReader(in)
	}
	// A helper that prompts needs the terminal it was launched from; anything else it
	// needs it can read from the environment it inherits.
	c.Env = os.Environ()

	if err := c.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("credential helper %q timed out after %s", cmd, helperTimeout)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if msg := strings.TrimSpace(errBuf.String()); msg != "" {
				return nil, fmt.Errorf("credential helper %q %s failed: %s", cmd, verb, msg)
			}
			return nil, fmt.Errorf("credential helper %q %s exited %d", cmd, verb, exitErr.ExitCode())
		}
		return nil, fmt.Errorf("run credential helper %q: %w", cmd, err)
	}
	return out.Bytes(), nil
}

func storeHelper(cmd string, c Credentials) error {
	data, err := json.Marshal(toStored(c))
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	_, err = runHelper(cmd, "store", data)
	return err
}

func loadHelper(cmd string) (Credentials, error) {
	out, err := runHelper(cmd, "get", nil)
	if err != nil {
		// A helper with no entry yet exits non-zero, and that is the ordinary first-run
		// case rather than a fault. Report it as "nothing stored" so Load can fall
		// through to the next tier.
		return Credentials{}, ErrNoCredentials
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return Credentials{}, ErrNoCredentials
	}

	var s stored
	if err := json.Unmarshal(out, &s); err != nil {
		return Credentials{}, fmt.Errorf("credential helper %q returned unparseable output", cmd)
	}

	c := s.creds()
	if !c.Valid() {
		return Credentials{}, ErrNoCredentials
	}
	return c, nil
}

func clearHelper(cmd string) error {
	if _, err := runHelper(cmd, "erase", nil); err != nil {
		// Erasing what is not there is not a failure, and a helper has no way to say so
		// other than a non-zero exit. Sign-out must not be blocked by it.
		return nil
	}
	return nil
}
