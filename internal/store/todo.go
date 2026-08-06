package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// The todo list: problems the user means to get to.
//
// Kept in its own table rather than as a column on problems, because the two have
// different owners. The problems table is a cache of LeetCode's data and a re-sync
// rewrites it wholesale; a list someone curated by hand must not be collateral damage of
// a refresh. The FK is deliberately absent for the same reason — an entry may be added
// for a problem this machine has not synced yet, and it should survive until it is.

// TodoEntry is one item on the list.
type TodoEntry struct {
	Slug string
	Note string
	// AddedAt orders the list. Oldest first: the list is a queue, and something added
	// three weeks ago is the thing most in danger of being forgotten.
	AddedAt time.Time
}

// AddTodo puts a problem on the list, or updates its note.
//
// Adding something already listed does NOT reset its position — that would send the
// oldest item to the back of the queue for the sake of a typo correction.
func (s *Store) AddTodo(ctx context.Context, slug, note string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO todo (problem_slug, note, added_at) VALUES (?,?,?)
		ON CONFLICT(problem_slug) DO UPDATE SET note = excluded.note`,
		slug, note, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("add %s to the todo list: %w", slug, err)
	}
	return nil
}

// RemoveTodo takes a problem off the list. Removing something absent is not an error —
// the caller wanted it gone, and it is.
func (s *Store) RemoveTodo(ctx context.Context, slug string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM todo WHERE problem_slug = ?`, slug); err != nil {
		return fmt.Errorf("remove %s from the todo list: %w", slug, err)
	}
	return nil
}

// IsTodo reports whether a problem is on the list.
func (s *Store) IsTodo(ctx context.Context, slug string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM todo WHERE problem_slug = ?`, slug).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the todo list for %s: %w", slug, err)
	}
	return true, nil
}

// Todos returns the list, oldest first.
func (s *Store) Todos(ctx context.Context) ([]TodoEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT problem_slug, note, added_at FROM todo ORDER BY added_at, problem_slug`)
	if err != nil {
		return nil, fmt.Errorf("read the todo list: %w", err)
	}
	defer rows.Close()

	var out []TodoEntry
	for rows.Next() {
		var (
			e  TodoEntry
			at int64
		)
		if err := rows.Scan(&e.Slug, &e.Note, &at); err != nil {
			return nil, fmt.Errorf("scan todo entry: %w", err)
		}
		e.AddedAt = time.Unix(at, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}

// TodoSlugs returns the listed slugs as a set, for marking rows on the board without a
// query per row.
func (s *Store) TodoSlugs(ctx context.Context) (map[string]bool, error) {
	entries, err := s.Todos(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		out[e.Slug] = true
	}
	return out, nil
}
