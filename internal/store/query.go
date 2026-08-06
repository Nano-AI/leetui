package store

import (
	"context"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

// Filter narrows a board query. The zero value returns everything, ID-ordered.
type Filter struct {
	// Text is a full-text query over titles, tags, and fetched statements.
	Text string

	// Difficulty filters to any of "Easy", "Medium", "Hard". Empty means all.
	Difficulty []string

	// Status is "ac" (solved), "notac" (attempted), or "todo" (untouched).
	Status string

	// Tags requires ALL listed tag slugs.
	Tags []string

	// Companies matches ANY listed company slug (premium data, D-008).
	Companies []string

	// PaidOnly, when set, filters to premium-only or free problems.
	PaidOnly *bool

	// Sort is "id" (default), "title", "acrate", or "difficulty".
	// A text query always sorts by relevance and ignores this.
	Sort string

	Limit  int
	Offset int
}

// Query returns board rows matching the filter.
func (s *Store) Query(ctx context.Context, f Filter) ([]Row, error) {
	var (
		sb    strings.Builder
		args  []any
		where []string
	)

	sb.WriteString(`
		SELECT p.slug, p.frontend_id, p.numeric_id, p.title, p.difficulty, p.ac_rate,
		       p.paid_only, p.status, p.has_solution, p.has_video,
		       COALESCE((SELECT group_concat(tag_slug) FROM problem_tags WHERE problem_slug = p.slug), ''),
		       COALESCE((SELECT group_concat(DISTINCT company_slug) FROM problem_companies WHERE problem_slug = p.slug), ''),
		       (p.detail_synced_at > 0)
		FROM problems p`)

	// A digits-only query is a problem number, not prose. Full-text search cannot answer
	// "146" — the number is in a column, not in the indexed text — so route it to the
	// numeric column instead. Prefix matching keeps it useful while still typing.
	numeric := numericQuery(f.Text)
	match := ""
	if numeric == "" {
		match = ftsQuery(f.Text)
	}

	switch {
	case numeric != "":
		where = append(where, `CAST(p.numeric_id AS TEXT) LIKE ?`)
		args = append(args, numeric+"%")
	case match != "":
		sb.WriteString(` JOIN problems_fts fts ON fts.slug = p.slug`)
		where = append(where, `problems_fts MATCH ?`)
		args = append(args, match)
	}

	if len(f.Difficulty) > 0 {
		where = append(where, `p.difficulty IN (`+placeholders(len(f.Difficulty))+`)`)
		for _, d := range f.Difficulty {
			args = append(args, d)
		}
	}

	switch f.Status {
	case "ac", "notac":
		where = append(where, `p.status = ?`)
		args = append(args, f.Status)
	case "todo":
		where = append(where, `(p.status IS NULL OR p.status = '')`)
	}

	// All tags must be present, so one EXISTS per tag rather than an IN list.
	for _, t := range f.Tags {
		where = append(where, `EXISTS (SELECT 1 FROM problem_tags pt WHERE pt.problem_slug = p.slug AND pt.tag_slug = ?)`)
		args = append(args, t)
	}

	if len(f.Companies) > 0 {
		where = append(where, `EXISTS (SELECT 1 FROM problem_companies pc
			WHERE pc.problem_slug = p.slug AND pc.company_slug IN (`+placeholders(len(f.Companies))+`))`)
		for _, c := range f.Companies {
			args = append(args, c)
		}
	}

	if f.PaidOnly != nil {
		where = append(where, `p.paid_only = ?`)
		args = append(args, *f.PaidOnly)
	}

	if len(where) > 0 {
		sb.WriteString(" WHERE " + strings.Join(where, " AND "))
	}

	orderClause, orderArgs := orderBy(f, match != "", numeric)
	sb.WriteString(" ORDER BY " + orderClause)
	args = append(args, orderArgs...)

	limit := f.Limit
	if limit <= 0 {
		limit = 500
	}
	sb.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, limit, f.Offset)

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query problems: %w", err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var (
			r         Row
			tagCSV    string
			compCSV   string
			hasDetail bool
		)
		if err := rows.Scan(
			&r.Slug, &r.FrontendID, &r.NumericID, &r.Title, &r.Difficulty, &r.AcRate,
			&r.PaidOnly, &r.Status, &r.HasSolution, &r.HasVideo,
			&tagCSV, &compCSV, &hasDetail,
		); err != nil {
			return nil, fmt.Errorf("scan problem: %w", err)
		}
		r.Tags = splitCSV(tagCSV)
		r.Companies = splitCSV(compCSV)
		r.HasDetail = hasDetail
		out = append(out, r)
	}
	return out, rows.Err()
}

// numericQuery returns the digits of a problem-number query, or "" if the text is not a
// bare number.
func numericQuery(text string) string {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 4 {
		return ""
	}
	for _, r := range t {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return strings.TrimLeft(t, "0")
}

// orderBy picks the sort, returning the clause and any bind arguments it needs.
//
// It returns arguments rather than interpolating them: user text must never reach the
// SQL string, even when the caller has already validated it.
//
// A text query sorts by relevance — someone who typed a query wants the best match
// first, not the lowest problem number.
func orderBy(f Filter, searching bool, numeric string) (string, []any) {
	if numeric != "" {
		// Exact number first, then prefix matches in numeric order: typing "1" should
		// put problem 1 above problem 1000.
		return "CASE WHEN CAST(p.numeric_id AS TEXT) = ? THEN 0 ELSE 1 END, p.numeric_id",
			[]any{numeric}
	}
	if searching {
		return "fts.rank, p.numeric_id", nil
	}
	switch f.Sort {
	case "title":
		return "p.title COLLATE NOCASE", nil
	case "acrate":
		return "p.ac_rate DESC", nil
	case "difficulty":
		// Textual order would give Easy, Hard, Medium. Sort by actual difficulty.
		return `CASE p.difficulty WHEN 'Easy' THEN 0 WHEN 'Medium' THEN 1 ELSE 2 END, p.numeric_id`, nil
	default:
		return "p.numeric_id", nil
	}
}
