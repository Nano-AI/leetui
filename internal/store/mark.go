package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Importance marks (D-032)
// ---------------------------------------------------------------------------
//
// "Was this worth doing?" — recorded per problem so a second pass through a study plan
// can skip what did not teach anything and revisit what did.
//
// Deliberately NOT the todo list. A todo is a queue you drain, and finishing a problem
// takes it off. A mark outlives the solve: "solved, and worth doing again" is the state
// this exists for, and the todo list has no way to say it.
//
// Ternary rather than a score. Up, down, or no opinion. A stacking counter would make
// every write a read-modify-write, which is exactly the race D-022 avoided by making
// every todo operation idempotent.

// Mark is a verdict on a problem.
type Mark string

const (
	// MarkNone is no opinion — the default, and what clearing returns a problem to.
	MarkNone Mark = ""

	// MarkUp is "worth doing again".
	MarkUp Mark = "up"

	// MarkDown is "not worth another pass".
	MarkDown Mark = "down"

	// MarkAny is a FILTER value meaning "marked either way". It is never stored, and
	// Valid rejects it, so it cannot reach the marks table by accident.
	MarkAny Mark = "any"
)

// Valid reports whether m is a mark the store will write. MarkNone is not: clearing goes
// through ClearMark, so a caller cannot half-set something by passing an empty string.
func (m Mark) Valid() bool { return m == MarkUp || m == MarkDown }

// Label is the mark in prose, for a status line or the CLI.
func (m Mark) Label() string {
	switch m {
	case MarkUp:
		return "important"
	case MarkDown:
		return "unimportant"
	default:
		return "unmarked"
	}
}

// MarkEntry is one marked problem.
type MarkEntry struct {
	Slug string
	Mark Mark
	Note string

	// MarkedAt orders the list, newest first — the opposite of the todo list. A todo is a
	// queue where the oldest item is most at risk of being forgotten; a mark is a
	// judgment, and the most recent one is the one you are still acting on.
	MarkedAt time.Time
}

// SetMark records a verdict, replacing any previous one.
//
// Idempotent: marking something twice is not an error. Re-marking DOES move it to the
// front, unlike re-adding a todo — a fresh judgment is newer information, and the list is
// ordered by recency rather than being a queue.
//
// An empty note does not erase an existing one. An agent that marks in bulk and a human
// who wrote a reason should not be in a race where the bulk pass wins.
func (s *Store) SetMark(ctx context.Context, slug string, mark Mark, note string) error {
	if !mark.Valid() {
		return fmt.Errorf("mark %s: %q is not up or down", slug, mark)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO marks (problem_slug, mark, note, marked_at) VALUES (?,?,?,?)
		ON CONFLICT(problem_slug) DO UPDATE SET
			mark      = excluded.mark,
			note      = CASE WHEN excluded.note = '' THEN marks.note ELSE excluded.note END,
			marked_at = excluded.marked_at`,
		slug, string(mark), note, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("mark %s as %s: %w", slug, mark, err)
	}
	return nil
}

// ClearMark returns a problem to no opinion. Clearing something unmarked is not an error —
// the caller wanted it gone, and it is.
func (s *Store) ClearMark(ctx context.Context, slug string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM marks WHERE problem_slug = ?`, slug); err != nil {
		return fmt.Errorf("clear the mark on %s: %w", slug, err)
	}
	return nil
}

// MarkOf reads one problem's verdict. An unmarked problem is MarkNone, not an error.
func (s *Store) MarkOf(ctx context.Context, slug string) (Mark, error) {
	var m string
	err := s.db.QueryRowContext(ctx,
		`SELECT mark FROM marks WHERE problem_slug = ?`, slug).Scan(&m)
	if err == sql.ErrNoRows {
		return MarkNone, nil
	}
	if err != nil {
		return MarkNone, fmt.Errorf("read the mark on %s: %w", slug, err)
	}
	return Mark(m), nil
}

// Marks returns marked problems, newest first. A zero want returns both directions.
func (s *Store) Marks(ctx context.Context, want Mark) ([]MarkEntry, error) {
	sqlText := `SELECT problem_slug, mark, note, marked_at FROM marks`
	var args []any
	if want.Valid() {
		sqlText += ` WHERE mark = ?`
		args = append(args, string(want))
	}
	sqlText += ` ORDER BY marked_at DESC, problem_slug`

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("read the marks: %w", err)
	}
	defer rows.Close()

	var out []MarkEntry
	for rows.Next() {
		var (
			e  MarkEntry
			mk string
			at int64
		)
		if err := rows.Scan(&e.Slug, &mk, &e.Note, &at); err != nil {
			return nil, fmt.Errorf("scan a mark: %w", err)
		}
		e.Mark = Mark(mk)
		e.MarkedAt = time.Unix(at, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkMap returns every mark keyed by slug, so the board can render a column without a
// query per row. Same bargain as TodoSlugs.
func (s *Store) MarkMap(ctx context.Context) (map[string]Mark, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT problem_slug, mark FROM marks`)
	if err != nil {
		return nil, fmt.Errorf("read the marks: %w", err)
	}
	defer rows.Close()

	out := map[string]Mark{}
	for rows.Next() {
		var slug, mk string
		if err := rows.Scan(&slug, &mk); err != nil {
			return nil, fmt.Errorf("scan a mark: %w", err)
		}
		out[slug] = Mark(mk)
	}
	return out, rows.Err()
}
