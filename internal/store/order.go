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

// markRank floats important problems and sinks unimportant ones, in ANY sort (D-032b).
//
// Up first, unmarked in the middle, down last. Unmarked sits between the two rather than
// after them because "no opinion" genuinely is between "worth redoing" and "written off",
// and on a board of four thousand where a dozen are marked, sorting everything unmarked
// below the handful marked down would make the rule useless.
//
// A `-` is not a delete. The problem stays in the collection, stays searchable, and stays
// findable — it just stops competing for the top of the list with the ones still worth
// your time. That is the difference between "I have judged this" and "I have hidden it",
// and only the first is a judgment you can revise later.
//
// It carries no bind argument, which is what makes it safe to prepend to every clause
// below without disturbing the order the remaining placeholders bind in.
const markRank = `CASE (SELECT mk.mark FROM marks mk WHERE mk.problem_slug = p.slug)
	WHEN 'up' THEN 0 WHEN 'down' THEN 2 ELSE 1 END`

// orderBy picks the sort, returning the clause and any bind arguments it needs.
//
// Every sort is prefixed with markRank, so the rule holds whether the board is showing
// problem numbers, a company pack by frequency, a study plan in curriculum order, or
// search results by relevance. With nothing marked the prefix is constant for every row
// and changes no ordering at all.
func orderBy(f Filter, searching bool, numeric string) (string, []any) {
	clause, args := sortClause(f, searching, numeric)
	return markRank + ", " + clause, args
}

// sortClause is the sort the user actually asked for, before demotion is applied.
//
// It returns arguments rather than interpolating them: user text must never reach the
// SQL string, even when the caller has already validated it.
//
// A text query sorts by relevance — someone who typed a query wants the best match
// first, not the lowest problem number.
func sortClause(f Filter, searching bool, numeric string) (string, []any) {
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
	case "todo":
		// The list is a queue: oldest first, because something added three weeks ago is
		// the one most in danger of being forgotten.
		return `(SELECT t.added_at FROM todo t WHERE t.problem_slug = p.slug), p.numeric_id`, nil
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
	case "mark":
		// markRank already groups these; all this adds is recency within a group, so the
		// verdict you just recorded is easy to find again.
		return `(SELECT mk.marked_at FROM marks mk WHERE mk.problem_slug = p.slug) DESC,
			p.numeric_id`, nil
	case "plan":
		// A plan's running order IS the plan (D-031). Without a plan set there is no rank
		// to read, so fall through to ID rather than emit a clause that orders every row
		// identically — the same bargain "frequency" makes without a company.
		if f.Plan == "" {
			return "p.numeric_id", nil
		}
		return `(SELECT pp.plan_rank FROM problem_plans pp
			WHERE pp.problem_slug = p.slug AND pp.plan_slug = ?), p.numeric_id`,
			[]any{f.Plan}

	case "contest":
		// Credit IS the running order of a contest: the four problems are meant to be
		// read 3, 4, 5, 6. Without a contest set there is no credit to read, so fall
		// through to ID on the same terms "plan" and "frequency" do.
		if f.Contest == "" {
			return "p.numeric_id", nil
		}
		return `(SELECT pc.credit FROM problem_contests pc
			WHERE pc.problem_slug = p.slug AND pc.contest_slug = ?), p.numeric_id`,
			[]any{f.Contest}
	case "difficulty":
		// Textual order would give Easy, Hard, Medium. Sort by actual difficulty.
		return `CASE p.difficulty WHEN 'Easy' THEN 0 WHEN 'Medium' THEN 1 ELSE 2 END, p.numeric_id`, nil
	default:
		return "p.numeric_id", nil
	}
}
