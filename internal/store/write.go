package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// ---------------------------------------------------------------------------
// Writes
// ---------------------------------------------------------------------------

// UpsertSummaries writes a page of problem-list rows in one transaction.
//
// Existing detail columns (content, metaData, snippets) are preserved: a list sync must
// never wipe a statement that was already fetched, or every sync would undo the lazy
// detail cache.
func (s *Store) UpsertSummaries(ctx context.Context, items []leetcode.ProblemSummary) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	upsert, err := tx.PrepareContext(ctx, `
		INSERT INTO problems (
			slug, frontend_id, numeric_id, title, difficulty, ac_rate,
			paid_only, status, has_solution, has_video, synced_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			frontend_id  = excluded.frontend_id,
			numeric_id   = excluded.numeric_id,
			title        = excluded.title,
			difficulty   = excluded.difficulty,
			ac_rate      = excluded.ac_rate,
			paid_only    = excluded.paid_only,
			status       = excluded.status,
			has_solution = excluded.has_solution,
			has_video    = excluded.has_video,
			synced_at    = excluded.synced_at`)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer upsert.Close()

	tagIns, err := tx.PrepareContext(ctx,
		`INSERT INTO tags (slug, name) VALUES (?,?) ON CONFLICT(slug) DO UPDATE SET name = excluded.name`)
	if err != nil {
		return fmt.Errorf("prepare tag insert: %w", err)
	}
	defer tagIns.Close()

	linkIns, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO problem_tags (problem_slug, tag_slug) VALUES (?,?)`)
	if err != nil {
		return fmt.Errorf("prepare tag link: %w", err)
	}
	defer linkIns.Close()

	for _, it := range items {
		if _, err := upsert.ExecContext(ctx,
			it.Slug, it.FrontendID, it.NumericID(), it.Title, string(it.Difficulty),
			it.AcRate, it.PaidOnly, string(it.Status), it.HasSolution, it.HasVideoSolution, now,
		); err != nil {
			return fmt.Errorf("upsert %s: %w", it.Slug, err)
		}

		// Tags are replaced wholesale: LeetCode occasionally removes one, and a
		// merge-only update would leave the stale tag searchable forever.
		if _, err := tx.ExecContext(ctx, `DELETE FROM problem_tags WHERE problem_slug = ?`, it.Slug); err != nil {
			return fmt.Errorf("clear tags for %s: %w", it.Slug, err)
		}
		tagNames := make([]string, 0, len(it.Tags))
		for _, t := range it.Tags {
			if _, err := tagIns.ExecContext(ctx, t.Slug, t.Name); err != nil {
				return fmt.Errorf("insert tag %s: %w", t.Slug, err)
			}
			if _, err := linkIns.ExecContext(ctx, it.Slug, t.Slug); err != nil {
				return fmt.Errorf("link tag %s: %w", t.Slug, err)
			}
			tagNames = append(tagNames, t.Name)
		}

		if err := reindexTx(ctx, tx, it.Slug, it.Title, strings.Join(tagNames, " ")); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert: %w", err)
	}
	return nil
}

// SetDetail stores a fetched problem statement and its starter snippets.
func (s *Store) SetDetail(ctx context.Context, p *leetcode.Problem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set detail: %w", err)
	}
	defer tx.Rollback()

	// INSERT-or-update, not UPDATE.
	//
	// A plain UPDATE silently matches nothing when the problem has never been through a
	// list sync, and the snippet insert below then fails its foreign key with
	// "FOREIGN KEY constraint failed" — a message that names neither the problem nor
	// the real cause. That is exactly what `leetui todo add <slug>` does on a fresh
	// database, which is the first thing docs/AGENTS.md tells an agent to run.
	//
	// The row is seeded from what the detail query itself returns, so a problem reached
	// this way is a complete row rather than a stub the board would render blank.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO problems (
			slug, frontend_id, numeric_id, title, difficulty, ac_rate, paid_only,
			status, has_solution, has_video, synced_at,
			question_id, content, meta_data, sample_testcase, example_testcases,
			hints, detail_synced_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			question_id       = excluded.question_id,
			content           = excluded.content,
			meta_data         = excluded.meta_data,
			sample_testcase   = excluded.sample_testcase,
			example_testcases = excluded.example_testcases,
			hints             = excluded.hints,
			detail_synced_at  = excluded.detail_synced_at,
			-- Only fill these in when the existing row has nothing. A list sync knows
			-- the user's own status and acceptance rate; the detail query does not
			-- always, and overwriting a real value with a blank would lose it.
			title             = CASE WHEN problems.title = '' THEN excluded.title ELSE problems.title END,
			frontend_id       = CASE WHEN problems.frontend_id = '' THEN excluded.frontend_id ELSE problems.frontend_id END,
			numeric_id        = CASE WHEN problems.numeric_id = 0 THEN excluded.numeric_id ELSE problems.numeric_id END,
			difficulty        = CASE WHEN problems.difficulty = '' THEN excluded.difficulty ELSE problems.difficulty END,
			paid_only         = CASE WHEN problems.synced_at = 0 THEN excluded.paid_only ELSE problems.paid_only END`,
		// The detail query does not return acceptance rate, the user's status, or
		// whether a solution exists — those come from the list sync. Zeroes here are
		// only ever written into a row that did not exist, and the CASE clauses above
		// stop them landing on one that did.
		p.Slug, p.FrontendID, p.NumericID(), p.Title, string(p.Difficulty), 0.0,
		p.PaidOnly, "", false, false, 0,
		p.QuestionID, p.Content, p.MetaData, p.SampleTestCase, p.ExampleTestcases,
		strings.Join(p.Hints, "\n\n"), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("store detail %s: %w", p.Slug, err)
	}

	for _, sn := range p.Snippets {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO snippets (problem_slug, lang_slug, lang, code) VALUES (?,?,?,?)
			ON CONFLICT(problem_slug, lang_slug) DO UPDATE SET
				lang = excluded.lang, code = excluded.code`,
			p.Slug, sn.LangSlug, sn.Lang, sn.Code); err != nil {
			return fmt.Errorf("store snippet %s/%s: %w", p.Slug, sn.LangSlug, err)
		}
	}

	// Reindex with the statement text so full-text search covers problem bodies.
	tagNames := make([]string, 0, len(p.Tags))
	for _, t := range p.Tags {
		tagNames = append(tagNames, t.Name)
	}
	body := stripHTML(p.Content) + " " + strings.Join(tagNames, " ")
	if err := reindexTx(ctx, tx, p.Slug, p.Title, body); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit detail: %w", err)
	}
	return nil
}

// reindexTx replaces a problem's FTS row.
func reindexTx(ctx context.Context, tx *sql.Tx, slug, title, body string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM problems_fts WHERE slug = ?`, slug); err != nil {
		return fmt.Errorf("clear fts for %s: %w", slug, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO problems_fts (slug, title, body) VALUES (?,?,?)`, slug, title, body); err != nil {
		return fmt.Errorf("index %s: %w", slug, err)
	}
	return nil
}

// stripHTML removes tags so the FTS index holds prose, not markup. It is deliberately
// crude — this feeds a search index, not a renderer (rendering is internal/render).
func stripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
			b.WriteRune(' ')
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}
