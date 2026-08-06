package auth

import (
	// Registers the "sqlite" driver used to read browser cookie databases.
	"database/sql"
	"io"
	"os"

	_ "modernc.org/sqlite"
)

// copyToTemp duplicates the cookie database so a running browser's lock does not block
// the read. The copy is deleted by the returned cleanup.
func copyToTemp(path string) (string, func(), error) {
	src, err := os.Open(path)
	if err != nil {
		return "", func() {}, err
	}
	defer src.Close()

	dst, err := os.CreateTemp("", "leetui-cookies-*.sqlite")
	if err != nil {
		return "", func() {}, err
	}
	name := dst.Name()
	cleanup := func() { os.Remove(name) }

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := dst.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	// A WAL sidecar holds recent writes; without it a freshly-created session can be
	// missing from the copy.
	for _, suffix := range []string{"-wal", "-shm"} {
		if data, err := os.ReadFile(path + suffix); err == nil {
			_ = os.WriteFile(name+suffix, data, 0o600)
			s := suffix
			prev := cleanup
			cleanup = func() { os.Remove(name + s); prev() }
		}
	}

	return name, cleanup, nil
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// openReadOnly opens a copied cookie database. The copy is ours alone, but read-only
// makes it impossible for a bug here to write to something that looks like a browser
// profile.
func openReadOnly(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path+"?mode=ro")
}
