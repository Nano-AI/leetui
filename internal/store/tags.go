package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// setPayloadTagsTx persists tags supplied by a detail or collection response.
// A nil slice means the field was omitted/null (contest details omit it entirely),
// so it must not erase known tags. A supplied list, even [], replaces the old set.
// The caller reindexes after this write, in the same transaction.
func setPayloadTagsTx(ctx context.Context, tx *sql.Tx, slug string, tags []leetcode.Tag) error {
	if tags == nil {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM problem_tags WHERE problem_slug = ?`, slug); err != nil {
		return fmt.Errorf("clear tags for %s: %w", slug, err)
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags (slug, name) VALUES (?,?)
			ON CONFLICT(slug) DO UPDATE SET name = excluded.name`, tag.Slug, tag.Name); err != nil {
			return fmt.Errorf("store tag %s for %s: %w", tag.Slug, slug, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO problem_tags (problem_slug, tag_slug)
			VALUES (?,?)`, slug, tag.Slug); err != nil {
			return fmt.Errorf("link tag %s to %s: %w", tag.Slug, slug, err)
		}
	}
	return nil
}
