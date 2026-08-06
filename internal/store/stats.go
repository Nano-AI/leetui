package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Count returns how many problems are stored.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM problems`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count problems: %w", err)
	}
	return n, nil
}

// Stats summarizes progress for the rail.
type Stats struct {
	Total              int
	Solved             int
	Attempted          int
	Easy, Medium, Hard int // solved, by difficulty
}

// Stats returns aggregate counts.
func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(status = 'ac'), 0),
			COALESCE(SUM(status = 'notac'), 0),
			COALESCE(SUM(status = 'ac' AND difficulty = 'Easy'), 0),
			COALESCE(SUM(status = 'ac' AND difficulty = 'Medium'), 0),
			COALESCE(SUM(status = 'ac' AND difficulty = 'Hard'), 0)
		FROM problems`).Scan(&st.Total, &st.Solved, &st.Attempted, &st.Easy, &st.Medium, &st.Hard)
	if err != nil {
		return st, fmt.Errorf("stats: %w", err)
	}
	return st, nil
}

// TagCounts returns every tag with how many problems carry it, most common first.
// Used to populate the filter UI without a second round of queries.
func (s *Store) TagCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag_slug, COUNT(*) FROM problem_tags GROUP BY tag_slug ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("tag counts: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var slug string
		var n int
		if err := rows.Scan(&slug, &n); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		out[slug] = n
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// Itoa is exposed for callers writing numeric checkpoints.
func Itoa(n int) string { return strconv.Itoa(n) }

// Atoi parses a checkpoint value, returning 0 for anything unparseable.
func Atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
