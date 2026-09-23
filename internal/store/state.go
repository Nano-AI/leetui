package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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
	KeyPlansSyncedAt     = "plans_synced_at"
	KeyContestsSyncedAt  = "contests_synced_at"
	KeyUsername          = "username"
	KeyIsPremium         = "is_premium"
)

// ContestKey is the sync_state key recording when a contest was last pulled.
//
// One key per contest. Unlike PlanKey this is written repeatedly during a live contest —
// re-pulling is how the questions appear at the start — so it records the last attempt,
// not a one-time fetch.
func ContestKey(contest string) string { return "contest:" + contest }

// PlanKey is the sync_state key recording when a study plan was last pulled.
//
// One key per plan, and no second axis: a plan has no timeframe. That is the whole
// difference from PackKey.
func PlanKey(plan string) string { return "plan:" + plan }

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

const upsertState = `INSERT INTO sync_state (key, value) VALUES (?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value`

// SetState writes a checkpoint value.
func (s *Store) SetState(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, upsertState, key, value)
	if err != nil {
		return fmt.Errorf("write sync state %q: %w", key, err)
	}
	return nil
}

// SetStates writes related state values atomically. A completion timestamp and
// cursor reset, or an account name and premium flag, must not be half-written.
func (s *Store) SetStates(ctx context.Context, values map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sync state write: %w", err)
	}
	defer tx.Rollback()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := tx.ExecContext(ctx, upsertState, key, values[key]); err != nil {
			return fmt.Errorf("write sync state %q: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sync state: %w", err)
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
