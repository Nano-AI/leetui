package store

import "strings"

// Sort and query-shape decisions for Query, kept apart from the clause assembly so
// each file reads as one job: query.go builds the statement, this decides its order.

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
	case "frequency":
		// Only companies carry a frequency, so without one this sort has nothing to read
		// and would order every row identically. Fall through to ID rather than emit a
		// clause that quietly does nothing.
		if len(f.Companies) == 0 {
			return "p.numeric_id", nil
		}
		args := make([]any, 0, len(f.Companies))
		for _, c := range f.Companies {
			args = append(args, c)
		}
		return `(SELECT MAX(pc.frequency) FROM problem_companies pc
			WHERE pc.problem_slug = p.slug
			  AND pc.company_slug IN (` + placeholders(len(f.Companies)) + `)) DESC, p.numeric_id`, args
	case "difficulty":
		// Textual order would give Easy, Hard, Medium. Sort by actual difficulty.
		return `CASE p.difficulty WHEN 'Easy' THEN 0 WHEN 'Medium' THEN 1 ELSE 2 END, p.numeric_id`, nil
	default:
		return "p.numeric_id", nil
	}
}
