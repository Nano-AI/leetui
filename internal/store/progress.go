package store

import (
	"context"
	"fmt"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// The user's own progress, as opposed to LeetCode's problem data.
//
// Written from two places that must not fight: the list sync, which reports what the site
// says, and a judgement arriving here in real time. Forward-only is what keeps them
// consistent when they disagree.

// SetStatus records the user's progress on one problem.
//
// The list sync is the only other writer of this column, so without this a solve stays
// invisible until the next full re-sync: you submit, the judge says Accepted, and the
// board still reads TRIED. That gap is exactly what the STATE column exists to close.
//
// Only ever moves progress FORWARD. A wrong answer on a problem already accepted must not
// downgrade it — LeetCode keeps the best result, and so does this.
func (s *Store) SetStatus(ctx context.Context, slug string, status leetcode.Status) error {
	if status == leetcode.StatusAccepted {
		_, err := s.db.ExecContext(ctx,
			`UPDATE problems SET status = ? WHERE slug = ?`, string(status), slug)
		if err != nil {
			return fmt.Errorf("set status %s: %w", slug, err)
		}
		return nil
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE problems SET status = ? WHERE slug = ? AND status <> ?`,
		string(status), slug, string(leetcode.StatusAccepted))
	if err != nil {
		return fmt.Errorf("set status %s: %w", slug, err)
	}
	return nil
}
