package store

import (
	// Registers the "sqlite" driver used by Open. Pure Go, no cgo (D-009).
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store is the local database. Safe for concurrent use; SQLite is in WAL mode and
// writes are expected to be serialized through the sync worker.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at dir/leetui.db and applies migrations.
func Open(dir string) (*Store, error) {
	path := filepath.Join(dir, "leetui.db")
	return OpenPath(path)
}

// OpenPath opens a specific file, or ":memory:" for tests.
func OpenPath(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		// WAL for concurrent readers during a sync; busy_timeout so a reader blocked by
		// the sync worker's write waits rather than erroring out under the user.
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite tolerates one writer. Keeping the pool small avoids spurious lock churn.
	db.SetMaxOpenConns(4)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the handle for packages that need raw access. Prefer adding a method here.
func (s *Store) DB() *sql.DB { return s.db }
