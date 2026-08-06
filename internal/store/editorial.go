package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// Editorial is a cached official solution.
type Editorial struct {
	Slug     string
	ID       string
	Title    string
	Content  string // markdown, not HTML — see leetcode.Solution
	PaidOnly bool

	// CanSee is false for an editorial that exists but is gated. The row is still stored:
	// knowing an editorial exists and is locked is worth a cache hit, and it stops the
	// pane re-requesting a gate on every keypress.
	CanSee   bool
	HasVideo bool
}

// SetEditorial caches an official solution.
//
// Gated editorials are stored with empty content and CanSee false, so the lock state is
// remembered rather than re-fetched. A later premium sign-in overwrites the row, because
// the fetch runs again once the pane is opened with a session that can read it.
func (s *Store) SetEditorial(ctx context.Context, slug string, sol *leetcode.Solution) error {
	if sol == nil {
		return fmt.Errorf("store editorial %s: no solution", slug)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO editorials
			(problem_slug, solution_id, title, content, paid_only, can_see, has_video, synced_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(problem_slug) DO UPDATE SET
			solution_id = excluded.solution_id,
			title       = excluded.title,
			content     = excluded.content,
			paid_only   = excluded.paid_only,
			can_see     = excluded.can_see,
			has_video   = excluded.has_video,
			synced_at   = excluded.synced_at`,
		slug, sol.ID, sol.Title, sol.Content, sol.PaidOnly, sol.CanSeeDetail,
		sol.HasVideoSolution, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("store editorial %s: %w", slug, err)
	}
	return nil
}

// GetEditorial reads a cached editorial. A miss returns ErrNotFound.
func (s *Store) GetEditorial(ctx context.Context, slug string) (*Editorial, error) {
	var e Editorial
	err := s.db.QueryRowContext(ctx, `
		SELECT problem_slug, solution_id, title, content, paid_only, can_see, has_video
		FROM editorials WHERE problem_slug = ?`, slug).Scan(
		&e.Slug, &e.ID, &e.Title, &e.Content, &e.PaidOnly, &e.CanSee, &e.HasVideo)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("editorial for %q: %w", slug, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get editorial %s: %w", slug, err)
	}
	return &e, nil
}
