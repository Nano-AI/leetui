package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// ---------------------------------------------------------------------------
// Study plans (D-031)
// ---------------------------------------------------------------------------

// Plan is a registry entry plus what is known locally about it.
type Plan struct {
	Slug string
	Name string

	// Highlight is LeetCode's own one-line pitch. Without it "LeetCode 75" and
	// "Top 100 Liked" are two names with nothing to choose between them.
	Highlight string

	// QuestionNum is the plan's size as LeetCode reports it, available before anything
	// has been pulled.
	QuestionNum int

	PremiumOnly bool

	// Stored is how many of the plan's problems are held locally. Zero means the plan has
	// never been pulled, which the picker shows as an invitation rather than as an empty
	// plan.
	Stored int

	// Solved is how many of those are accepted. This is the number the picker leads with:
	// a plan is a thing you are partway through, and "18/150" is the only figure that
	// answers "where was I".
	Solved int
}

// Progress is the plan's completion as a fraction of LeetCode's own size, in percent.
//
// It divides by QuestionNum rather than by Stored on purpose: dividing by what happens to
// be downloaded would report 100% for a plan that is one problem synced and one problem
// solved.
func (p Plan) Progress() int {
	if p.QuestionNum <= 0 {
		return 0
	}
	return p.Solved * 100 / p.QuestionNum
}

// UpsertPlans merges registry entries.
//
// Unlike the company registry this MERGES rather than replaces. The registry is assembled
// from a seed list unioned with a tag sweep (D-031), and neither is authoritative on its
// own — a wholesale replace would let a tag sweep that returned less than usual delete
// plans the seed list knows perfectly well.
//
// A zero question_num is not written over a known one, because a brief from a tag sweep
// is allowed to be less complete than a detail fetch.
func (s *Store) UpsertPlans(ctx context.Context, items []leetcode.PlanBrief) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin plan upsert: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO study_plans (slug, name, highlight, question_num, premium_only)
		VALUES (?,?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			name         = excluded.name,
			highlight    = excluded.highlight,
			question_num = CASE WHEN excluded.question_num > 0 THEN excluded.question_num ELSE study_plans.question_num END,
			premium_only = excluded.premium_only`)
	if err != nil {
		return fmt.Errorf("prepare plan upsert: %w", err)
	}
	defer stmt.Close()

	for _, p := range items {
		if p.Slug == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, p.Slug, p.Name, p.Highlight,
			p.QuestionNum, p.PremiumOnly); err != nil {
			return fmt.Errorf("upsert plan %s: %w", p.Slug, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit plan upsert: %w", err)
	}
	return nil
}

// Plans returns the registry, optionally filtered by a substring of the name or slug.
//
// Ordering puts plans you have started first, then the rest by size. A plan in progress is
// the one you came here to resume; the website leads with the same thing under "Ongoing".
func (s *Store) Plans(ctx context.Context, query string) ([]Plan, error) {
	sql := `
		SELECT sp.slug, sp.name, sp.highlight, sp.question_num, sp.premium_only,
		       COALESCE((SELECT COUNT(*) FROM problem_plans pp
		                 WHERE pp.plan_slug = sp.slug), 0) AS stored,
		       COALESCE((SELECT COUNT(*) FROM problem_plans pp
		                 JOIN problems p ON p.slug = pp.problem_slug
		                 WHERE pp.plan_slug = sp.slug AND p.status = 'ac'), 0) AS solved
		FROM study_plans sp`
	var args []any

	if q := strings.TrimSpace(strings.ToLower(query)); q != "" {
		sql += ` WHERE LOWER(sp.name) LIKE ? OR LOWER(sp.slug) LIKE ?`
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	sql += ` ORDER BY (solved > 0) DESC, sp.question_num DESC, sp.name COLLATE NOCASE`

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query study plans: %w", err)
	}
	defer rows.Close()

	var out []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.Slug, &p.Name, &p.Highlight, &p.QuestionNum,
			&p.PremiumOnly, &p.Stored, &p.Solved); err != nil {
			return nil, fmt.Errorf("scan study plan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPlan replaces one plan's problem list.
//
// Order is the payload here. Rank is the question's position in the flattened plan, so the
// board can reproduce the curriculum — Array/String before Two Pointers before Sliding
// Window — which is the entire reason a plan beats a tag filter.
//
// Replacing rather than merging means a problem dropped from the plan leaves the list. The
// write is one transaction, so a cancelled sync leaves the previous plan intact.
//
// Problems not already known locally are inserted as bare rows, for the same reason packs
// do it: a plan is a place premium problems surface before a full list sync reaches them.
func (s *Store) SetPlan(ctx context.Context, plan string, p leetcode.Plan) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin plan write: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM problem_plans WHERE plan_slug = ?`, plan); err != nil {
		return fmt.Errorf("clear plan %s: %w", plan, err)
	}

	seed, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO problems
			(slug, frontend_id, numeric_id, title, difficulty, paid_only, status)
		VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("prepare problem seed: %w", err)
	}
	defer seed.Close()

	link, err := tx.PrepareContext(ctx, `
		INSERT INTO problem_plans (problem_slug, plan_slug, plan_rank, group_name)
		VALUES (?,?,?,?)
		ON CONFLICT(problem_slug, plan_slug) DO UPDATE SET
			plan_rank  = excluded.plan_rank,
			group_name = excluded.group_name`)
	if err != nil {
		return fmt.Errorf("prepare plan link: %w", err)
	}
	defer link.Close()

	rank := 0
	for _, g := range p.Groups {
		for _, q := range g.Questions {
			if q.Slug == "" {
				continue
			}
			sum := q.Summary()
			if _, err := seed.ExecContext(ctx, sum.Slug, sum.FrontendID, sum.NumericID(),
				sum.Title, string(sum.Difficulty), sum.PaidOnly, string(sum.Status)); err != nil {
				return fmt.Errorf("seed problem %s: %w", q.Slug, err)
			}
			if _, err := link.ExecContext(ctx, q.Slug, plan, rank, g.Name); err != nil {
				return fmt.Errorf("link %s to %s: %w", q.Slug, plan, err)
			}
			if err := setPayloadTagsTx(ctx, tx, q.Slug, q.Tags); err != nil {
				return err
			}
			if err := reindexTx(ctx, tx, q.Slug); err != nil {
				return err
			}
			rank++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit plan %s: %w", plan, err)
	}
	return nil
}

// PlanCount returns how many problems are stored for a plan.
func (s *Store) PlanCount(ctx context.Context, plan string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM problem_plans WHERE plan_slug = ?`, plan).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count plan %s: %w", plan, err)
	}
	return n, nil
}

// PlanGroups returns each stored problem's chapter within a plan, so the board can label
// rows without a query per row.
func (s *Store) PlanGroups(ctx context.Context, plan string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT problem_slug, group_name FROM problem_plans WHERE plan_slug = ?`, plan)
	if err != nil {
		return nil, fmt.Errorf("query plan groups %s: %w", plan, err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var slug, group string
		if err := rows.Scan(&slug, &group); err != nil {
			return nil, fmt.Errorf("scan plan group: %w", err)
		}
		out[slug] = group
	}
	return out, rows.Err()
}
