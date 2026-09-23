package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// contestOf builds a contest fixture, so a test can say what it expects without restating
// the wire shape every time.
func contestOf(slug, title string, start int64, qs ...leetcode.ContestQuestion) *leetcode.Contest {
	return &leetcode.Contest{
		ContestBrief: leetcode.ContestBrief{
			Title: title, Slug: slug, StartTime: start, Duration: 5400,
		},
		Questions: qs,
	}
}

func cq(slug, id string, credit int) leetcode.ContestQuestion {
	return leetcode.ContestQuestion{
		QuestionID: id, Title: slug, Slug: slug, Credit: credit,
	}
}

// TestSetContestKeepsCreditOrder is why a contest sorts on credit rather than id.
//
// The four problems are meant to be read 3, 4, 5, 6, and their ids arrive in whatever
// order LeetCode assigned them.
func TestSetContestKeepsCreditOrder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.SetContest(ctx, contestOf("weekly-517", "Weekly 517", 1788057000,
		cq("hard-one", "4294", 6), cq("easy-one", "4295", 3), cq("mid-one", "4297", 4)))
	if err != nil {
		t.Fatalf("set contest: %v", err)
	}

	qs, err := s.ContestQuestions(ctx, "weekly-517")
	if err != nil {
		t.Fatalf("read questions: %v", err)
	}
	want := []string{"easy-one", "mid-one", "hard-one"}
	if len(qs) != len(want) {
		t.Fatalf("got %d questions, want %d", len(qs), len(want))
	}
	for i, w := range want {
		if qs[i].Slug != w {
			t.Errorf("question %d = %s, want %s — credit order beats id order", i, qs[i].Slug, w)
		}
	}
}

// TestSetContestEmptyDoesNotWipe is the single most dangerous case in this feature.
//
// Before a contest starts LeetCode returns an EMPTY question list rather than an error.
// A refresh pressed during a live contest — which is the normal way to use this — would
// then erase the four problems the user is working on if empty were treated as truth.
func TestSetContestEmptyDoesNotWipe(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	full := contestOf("weekly-517", "Weekly 517", 1788057000,
		cq("a", "1", 3), cq("b", "2", 4))
	if err := s.SetContest(ctx, full); err != nil {
		t.Fatalf("set contest: %v", err)
	}

	// The same contest, answered empty — exactly what a pre-start fetch returns.
	empty := contestOf("weekly-517", "Weekly 517", 1788057000)
	if err := s.SetContest(ctx, empty); err != nil {
		t.Fatalf("set empty contest: %v", err)
	}

	qs, err := s.ContestQuestions(ctx, "weekly-517")
	if err != nil {
		t.Fatalf("read questions: %v", err)
	}
	if len(qs) != 2 {
		t.Fatalf("got %d questions after an empty refresh, want 2 — an empty answer wiped the contest", len(qs))
	}
}

// TestSetContestStoresQuestionID is what makes submitting during a contest possible.
//
// A live contest's problems are not in the problems table, so the contest's own record of
// the internal id is the only place it exists on this machine. Without it there is no
// submission.
func TestSetContestStoresQuestionID(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SetContest(ctx, contestOf("weekly-517", "Weekly 517", 1788057000,
		cq("a", "4295", 3))); err != nil {
		t.Fatalf("set contest: %v", err)
	}
	qs, err := s.ContestQuestions(ctx, "weekly-517")
	if err != nil {
		t.Fatalf("read questions: %v", err)
	}
	if len(qs) != 1 || qs[0].QuestionID != "4295" {
		t.Fatalf("questionId not stored: %+v — a contest submission cannot be built without it", qs)
	}
}

// TestContestProblemsAreSearchable is the bug a live contest would have hit.
//
// A contest's problems are not in the problem set while it runs, so they exist only as
// rows SetContest seeds. problems_fts is a standalone table that a plain INSERT into
// problems does not touch — so without an explicit reindex, typing a contest problem's
// own name on a contest-filtered board returns nothing at all.
func TestContestProblemsAreSearchable(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SetContest(ctx, contestOf("weekly-517", "Weekly 517", 1788057000,
		cq("maximize-fixed-points", "4294", 6))); err != nil {
		t.Fatalf("set contest: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Contest: "weekly-517", Text: "maximize"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("searching a contest problem by name gave %d rows, want 1 — "+
			"the seeded row never reached the FTS index", len(rows))
	}
}

// TestUpsertContestsMerges guards the schedule against the shrinking window it is built
// from: upcomingContests knows about one or two contests, and this table remembers every
// contest ever pulled. A replace would delete the past on every refresh.
func TestUpsertContestsMerges(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertContests(ctx, []leetcode.ContestBrief{
		{Title: "Weekly 500", Slug: "weekly-500", StartTime: 1000, Duration: 5400},
		{Title: "Weekly 501", Slug: "weekly-501", StartTime: 2000, Duration: 5400},
	}); err != nil {
		t.Fatalf("seed contests: %v", err)
	}
	// A later schedule refresh mentions only the new one.
	if err := s.UpsertContests(ctx, []leetcode.ContestBrief{
		{Title: "Weekly 502", Slug: "weekly-502", StartTime: 3000, Duration: 5400},
	}); err != nil {
		t.Fatalf("refresh contests: %v", err)
	}

	list, err := s.Contests(ctx, "")
	if err != nil {
		t.Fatalf("read contests: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("got %d contests, want 3 — a refresh deleted the ones it did not mention", len(list))
	}
	// Newest first: the next contest and the last one are both what you came for.
	if list[0].Slug != "weekly-502" {
		t.Errorf("first row = %s, want weekly-502 — the schedule sorts newest first", list[0].Slug)
	}
}

// TestUpsertContestsKeepsAKnownStart is the other half of the merge rule.
//
// A brief from a list is allowed to be less complete than a detail fetch, and a contest
// whose start time is forgotten is one whose countdown breaks.
func TestUpsertContestsKeepsAKnownStart(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertContests(ctx, []leetcode.ContestBrief{
		{Title: "Weekly 517", Slug: "weekly-517", StartTime: 1788057000, Duration: 5400},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.UpsertContests(ctx, []leetcode.ContestBrief{
		{Title: "Weekly 517", Slug: "weekly-517"}, // no start, no duration
	}); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	c, err := s.ContestOf(ctx, "weekly-517")
	if err != nil {
		t.Fatalf("read contest: %v", err)
	}
	if c.StartTime != 1788057000 || c.Duration != 5400 {
		t.Errorf("start/duration = %d/%d, want 1788057000/5400 — a bare brief overwrote a known start",
			c.StartTime, c.Duration)
	}
}

// TestContestFilterAndSort is the board's half: a contest narrows to its own problems and
// orders them by credit.
func TestContestFilterAndSort(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	if err := s.SetContest(ctx, contestOf("weekly-517", "Weekly 517", 1788057000,
		cq("trapping-rain-water", "42", 6), cq("two-sum", "1", 3))); err != nil {
		t.Fatalf("set contest: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Contest: "weekly-517", Sort: "contest"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 — the contest filter did not narrow the board", len(rows))
	}
	if rows[0].Slug != "two-sum" {
		t.Errorf("first row = %s, want two-sum — credit 3 comes before credit 6", rows[0].Slug)
	}
}

// TestContestSortNeedsAContest mirrors the plan rule: without a contest there is no credit
// to read, so the sort must fall through rather than order every row identically.
func TestContestSortNeedsAContest(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	rows, err := s.Query(ctx, Filter{Sort: "contest"})
	if err != nil {
		t.Fatalf("query with a contest sort and no contest: %v", err)
	}
	if len(rows) != len(sample()) {
		t.Fatalf("got %d rows, want %d", len(rows), len(sample()))
	}
}

// TestSetContestSolvedMarksOnlyItsOwn keeps one contest's progress out of another's.
func TestSetContestSolvedMarksOnlyItsOwn(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SetContest(ctx, contestOf("a", "A", 100, cq("shared", "1", 3))); err != nil {
		t.Fatalf("set a: %v", err)
	}
	if err := s.SetContest(ctx, contestOf("b", "B", 200, cq("shared", "1", 3))); err != nil {
		t.Fatalf("set b: %v", err)
	}
	if err := s.SetContestSolved(ctx, "a", "shared"); err != nil {
		t.Fatalf("mark solved: %v", err)
	}

	ca, err := s.ContestOf(ctx, "a")
	if err != nil {
		t.Fatalf("read a: %v", err)
	}
	cb, err := s.ContestOf(ctx, "b")
	if err != nil {
		t.Fatalf("read b: %v", err)
	}
	if ca.Solved != 1 {
		t.Errorf("contest a solved = %d, want 1", ca.Solved)
	}
	if cb.Solved != 0 {
		t.Errorf("contest b solved = %d, want 0 — solving in one contest credited another", cb.Solved)
	}
}
