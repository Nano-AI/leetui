package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Get returns a problem's full record.
func (s *Store) Get(ctx context.Context, slug string) (*Detail, error) {
	var (
		d       Detail
		tagCSV  string
		compCSV string
		hints   string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT p.slug, p.frontend_id, p.numeric_id, p.title, p.difficulty, p.ac_rate,
		       p.paid_only, p.status, p.has_solution, p.has_video,
		       COALESCE((SELECT group_concat(tag_slug) FROM problem_tags WHERE problem_slug = p.slug), ''),
		       COALESCE((SELECT group_concat(DISTINCT company_slug) FROM problem_companies WHERE problem_slug = p.slug), ''),
		       (p.detail_synced_at > 0),
		       p.question_id, p.content, p.meta_data, p.sample_testcase, p.example_testcases, p.hints
		FROM problems p WHERE p.slug = ?`, slug).Scan(
		&d.Slug, &d.FrontendID, &d.NumericID, &d.Title, &d.Difficulty, &d.AcRate,
		&d.PaidOnly, &d.Status, &d.HasSolution, &d.HasVideo,
		&tagCSV, &compCSV, &d.HasDetail,
		&d.QuestionID, &d.Content, &d.MetaData, &d.SampleTestCase, &d.ExampleTestcases, &hints,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%q: %w", slug, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", slug, err)
	}

	d.Tags = splitCSV(tagCSV)
	d.Companies = splitCSV(compCSV)
	if hints != "" {
		d.Hints = strings.Split(hints, "\n\n")
	}

	d.Snippets = map[string]string{}
	rows, err := s.db.QueryContext(ctx, `SELECT lang_slug, code FROM snippets WHERE problem_slug = ?`, slug)
	if err != nil {
		return nil, fmt.Errorf("get snippets %s: %w", slug, err)
	}
	defer rows.Close()
	for rows.Next() {
		var lang, code string
		if err := rows.Scan(&lang, &code); err != nil {
			return nil, fmt.Errorf("scan snippet: %w", err)
		}
		d.Snippets[lang] = code
	}
	return &d, rows.Err()
}
