package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Nano-AI/leetui/internal/config"
)

// FileName is the credentials file, in leetui's config dir.
//
// Named for what it holds rather than something coy, because a user who finds it should
// immediately understand what they have found. doctor prints the full path.
const FileName = "credentials.json"

// fileMode is owner read/write and nothing else. Enforced on write AND on read.
const fileMode fs.FileMode = 0o600

// FilePath is where the file backend keeps credentials.
//
// Built on config.Dir, which already honours LEETUI_CONFIG_DIR, so tests get isolation
// for free and a second profile costs one environment variable.
func FilePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

func storeFile(c Credentials) error {
	path, err := FilePath()
	if err != nil {
		return err
	}

	// The directory matters as much as the file: a world-writable config dir means
	// somebody else can swap the credentials out from under us.
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("secure config dir: %w", err)
	}

	data, err := json.MarshalIndent(toStored(c), "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	data = append(data, '\n')

	// Temp file plus rename, so an interrupted write cannot leave a half-written secret
	// where a whole one used to be. Created in the destination directory so the rename
	// stays on one filesystem, and created at 0600 rather than chmod'ed after: the gap
	// between the two is a window where the session is world-readable.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if err := tmp.Chmod(fileMode); err != nil {
		tmp.Close()
		return fmt.Errorf("secure credentials file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

func loadFile() (Credentials, error) {
	var c Credentials

	path, err := FilePath()
	if err != nil {
		return c, err
	}

	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, ErrNoCredentials
	}
	if err != nil {
		return c, fmt.Errorf("read credentials: %w", err)
	}

	// Refuse rather than repair. A file that is group- or world-readable has already
	// been exposed for however long the mode has been wrong, and silently chmod'ing it
	// would hide that from the only person who can decide what to do about it.
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return c, fmt.Errorf("%s is readable by others (mode %04o); run: chmod 600 %s", path, perm, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("read credentials: %w", err)
	}

	var s stored
	if err := json.Unmarshal(data, &s); err != nil {
		// Never quote the file's contents into the error: it is a session token.
		return c, fmt.Errorf("parse %s: %w", path, err)
	}

	c = s.creds()
	if !c.Valid() {
		return Credentials{}, ErrNoCredentials
	}
	return c, nil
}

func clearFile() error {
	path, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
