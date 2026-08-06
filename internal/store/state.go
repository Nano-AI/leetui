package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Sync state
// ---------------------------------------------------------------------------

// Sync state keys.
const (
	KeyProblemsSyncedAt  = "problems_synced_at"
	KeyProblemsTotal     = "problems_total"
	KeyProblemsCursor    = "problems_cursor" // resume offset for an interrupted sync
	KeyCompaniesSyncedAt = "companies_synced_at"
	KeyUsername          = "username"
	KeyIsPremium         = "is_premium"
)

// PackKey is the sync_state key recording when a company pack was last pulled.
//
// One key per company AND timeframe, because they are refreshed independently: "last 30
// days" goes stale in weeks and "all time" barely moves.
func PackKey(company, timeframe string) string {
	return "pack:" + company + ":" + timeframe
}

// GetState reads a checkpoint value. A missing key returns "" and no error.
func (s *Store) GetState(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM sync_state WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read sync state %q: %w", key, err)
	}
	return v, nil
}

// SetState writes a checkpoint value.
func (s *Store) SetState(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sync_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("write sync state %q: %w", key, err)
	}
	return nil
}

// SyncedAt returns when the problem list last completed a full sync, zero if never.
func (s *Store) SyncedAt(ctx context.Context) (time.Time, error) {
	v, err := s.GetState(ctx, KeyProblemsSyncedAt)
	if err != nil || v == "" {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, v)
}
