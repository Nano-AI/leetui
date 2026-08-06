package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// ---------------------------------------------------------------------------
// Company packs (D-006)
// ---------------------------------------------------------------------------

// Company is a registry entry plus what is known locally about it.
type Company struct {
	Slug string
	Name string

	// QuestionCount is LeetCode's all-time figure, from the registry.
	QuestionCount int

	// Synced is how many of this company's problems are stored locally, across every
	// timeframe. Zero means the pack has never been pulled — which the browse list shows
	// as an invitation to sync rather than as an empty company.
	Synced int
}

// UpsertCompanies replaces the company registry.
//
// The registry is one request and readable signed out, so it is refreshed wholesale
// rather than merged. Existing problem_companies rows are untouched: a company dropping
// out of the registry should not silently erase a pack the user already pulled.
func (s *Store) UpsertCompanies(ctx context.Context, items []leetcode.Company) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin company upsert: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO companies (slug, name, question_count) VALUES (?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			name           = excluded.name,
			question_count = excluded.question_count`)
	if err != nil {
		return fmt.Errorf("prepare company upsert: %w", err)
	}
	defer stmt.Close()

	for _, c := range items {
		if c.Slug == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, c.Slug, c.Name, c.QuestionCount); err != nil {
			return fmt.Errorf("upsert company %s: %w", c.Slug, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit company upsert: %w", err)
	}
	return nil
}

// Companies returns the registry, most-problems first, optionally filtered by a
// substring of the name or slug.
//
// Ordering by size rather than alphabetically puts the companies people actually prepare
// for at the top of a 984-row list.
func (s *Store) Companies(ctx context.Context, query string) ([]Company, error) {
	sql := `
		SELECT c.slug, c.name, c.question_count,
		       COALESCE((SELECT COUNT(DISTINCT problem_slug) FROM problem_companies
		                 WHERE company_slug = c.slug), 0)
		FROM companies c`
	var args []any

	if q := strings.TrimSpace(strings.ToLower(query)); q != "" {
		sql += ` WHERE LOWER(c.name) LIKE ? OR LOWER(c.slug) LIKE ?`
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	sql += ` ORDER BY c.question_count DESC, c.name COLLATE NOCASE`

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	var out []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.Slug, &c.Name, &c.QuestionCount, &c.Synced); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetPack replaces one company/timeframe slice of problem_companies.
//
// Replacing rather than merging is deliberate: a re-sync should reflect the pack as it
// is now, and a problem the company stopped asking must leave the list. The delete is
// scoped to this timeframe, so refreshing "last 30 days" cannot wipe "all time".
//
// Problems not already known locally are inserted as bare rows. A company pack is the
// only place some premium problems surface before a full list sync has run, and a
// pack that silently dropped half its entries would be worse than useless.
func (s *Store) SetPack(ctx context.Context, company string, tf string, items []leetcode.PackQuestion) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin pack write: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM problem_companies WHERE company_slug = ? AND timeframe = ?`,
		company, tf); err != nil {
		return fmt.Errorf("clear pack %s/%s: %w", company, tf, err)
	}

	seed, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO problems
			(slug, frontend_id, numeric_id, title, difficulty, ac_rate, paid_only, status)
		VALUES (?,?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("prepare problem seed: %w", err)
	}
	defer seed.Close()

	link, err := tx.PrepareContext(ctx, `
		INSERT INTO problem_companies (problem_slug, company_slug, frequency, timeframe)
		VALUES (?,?,?,?)
		ON CONFLICT(problem_slug, company_slug, timeframe) DO UPDATE SET
			frequency = excluded.frequency`)
	if err != nil {
		return fmt.Errorf("prepare pack link: %w", err)
	}
	defer link.Close()

	for _, q := range items {
		if q.Slug == "" {
			continue
		}
		sum := q.Summary()
		if _, err := seed.ExecContext(ctx, sum.Slug, sum.FrontendID, sum.NumericID(),
			sum.Title, string(sum.Difficulty), sum.AcRate, sum.PaidOnly, string(sum.Status)); err != nil {
			return fmt.Errorf("seed problem %s: %w", q.Slug, err)
		}
		if _, err := link.ExecContext(ctx, q.Slug, company, q.Frequency, tf); err != nil {
			return fmt.Errorf("link %s to %s: %w", q.Slug, company, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit pack %s/%s: %w", company, tf, err)
	}
	return nil
}

// PackCount returns how many problems are stored for a company and timeframe.
func (s *Store) PackCount(ctx context.Context, company, tf string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM problem_companies WHERE company_slug = ? AND timeframe = ?`,
		company, tf).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count pack %s/%s: %w", company, tf, err)
	}
	return n, nil
}
