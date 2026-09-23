package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// planOf builds a two-chapter plan out of slugs, so a test can say what order it expects
// without restating the wire shape every time.
func planOf(slug, name string, chapters ...[]string) leetcode.Plan {
	p := leetcode.Plan{PlanBrief: leetcode.PlanBrief{Slug: slug, Name: name}}
	chapterNames := []string{"Array / String", "Two Pointers", "Sliding Window"}
	for i, c := range chapters {
		g := leetcode.PlanGroup{Name: chapterNames[i%len(chapterNames)]}
		for _, s := range c {
			g.Questions = append(g.Questions, leetcode.PlanQuestion{
				Slug: s, Title: s, Difficulty: leetcode.Medium,
			})
			p.QuestionNum++
		}
		g.QuestionNum = len(c)
		p.Groups = append(p.Groups, g)
	}
	return p
}

func seedPlans(t *testing.T, s *Store) {
	t.Helper()
	err := s.UpsertPlans(context.Background(), []leetcode.PlanBrief{
		{Slug: "top-interview-150", Name: "Top Interview 150",
			Highlight: "Must-do List for Interview Prep", QuestionNum: 150},
		{Slug: "leetcode-75", Name: "LeetCode 75",
			Highlight: "Ace Coding Interview with 75 Qs", QuestionNum: 75},
		{Slug: "premium-algo-100", Name: "Premium Algo 100",
			QuestionNum: 100, PremiumOnly: true},
	})
	if err != nil {
		t.Fatalf("upsert plans: %v", err)
	}
}

// TestSetPlanKeepsCurriculumOrder is the whole point of a plan: the board must reproduce
// the author's running order, not the problem numbers.
func TestSetPlanKeepsCurriculumOrder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	// Deliberately descending by ID, so ID order and plan order disagree.
	plan := planOf("top-interview-150", "Top Interview 150",
		[]string{"lru-cache", "two-sum"},
		[]string{"trapping-rain-water"})
	if err := s.SetPlan(ctx, "top-interview-150", plan); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Plan: "top-interview-150", Sort: "plan"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []string{"lru-cache", "two-sum", "trapping-rain-water"}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		if rows[i].Slug != w {
			t.Errorf("position %d is %q, want %q — plan order beats ID order", i, rows[i].Slug, w)
		}
	}
}

// TestPlanSortNeedsAPlan: "plan" with no plan set has no rank to read and must fall back
// to ID rather than emit a clause that orders every row identically.
func TestPlanSortNeedsAPlan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Sort: "plan"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 3 || rows[0].Slug != "two-sum" {
		t.Fatalf("plan sort without a plan gave %+v, want ID order starting at two-sum", rows)
	}
}

func TestSetPlanSeedsUnknownProblems(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)

	plan := leetcode.Plan{
		PlanBrief: leetcode.PlanBrief{Slug: "leetcode-75", Name: "LeetCode 75", QuestionNum: 1},
		Groups: []leetcode.PlanGroup{{Name: "Array / String", Questions: []leetcode.PlanQuestion{
			{Slug: "meeting-rooms-ii", Title: "Meeting Rooms II", FrontendID: "253",
				Difficulty: leetcode.Medium, PaidOnly: true},
		}}},
	}
	if err := s.SetPlan(ctx, "leetcode-75", plan); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Plan: "leetcode-75"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0].Title != "Meeting Rooms II" {
		t.Fatalf("plan did not seed the unknown problem: %+v", rows)
	}
	if !rows[0].PaidOnly {
		t.Error("seeded problem lost its premium flag")
	}
}

// TestSetPlanReplaces: a problem dropped from a plan must leave, and another plan holding
// the same problem must not be touched.
func TestSetPlanReplaces(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	full := planOf("top-interview-150", "Top Interview 150", []string{"two-sum", "lru-cache"})
	if err := s.SetPlan(ctx, "top-interview-150", full); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}
	if err := s.SetPlan(ctx, "leetcode-75", full); err != nil {
		t.Fatalf("SetPlan other: %v", err)
	}

	trimmed := planOf("top-interview-150", "Top Interview 150", []string{"two-sum"})
	if err := s.SetPlan(ctx, "top-interview-150", trimmed); err != nil {
		t.Fatalf("SetPlan refresh: %v", err)
	}

	n, err := s.PlanCount(ctx, "top-interview-150")
	if err != nil || n != 1 {
		t.Errorf("plan has %d problems after refresh (err %v), want 1", n, err)
	}
	if n, err = s.PlanCount(ctx, "leetcode-75"); err != nil || n != 2 {
		t.Errorf("other plan is %d (err %v) — refreshing one plan wiped another", n, err)
	}
}

// TestPlanProgressCountsSolved is what the picker leads with. two-sum is accepted in
// sample(), the others are not.
func TestPlanProgressCountsSolved(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	plan := planOf("leetcode-75", "LeetCode 75", []string{"two-sum", "lru-cache"})
	if err := s.SetPlan(ctx, "leetcode-75", plan); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}

	plans, err := s.Plans(ctx, "leetcode-75")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	got := plans[0]
	if got.Stored != 2 {
		t.Errorf("Stored = %d, want 2", got.Stored)
	}
	if got.Solved != 1 {
		t.Errorf("Solved = %d, want 1 (only two-sum is accepted)", got.Solved)
	}
	// 1 of the plan's stated 75, not 1 of the 2 that happen to be downloaded. Dividing by
	// what is local would report 50% for a plan barely begun.
	if got.Progress() != 1 {
		t.Errorf("Progress = %d%%, want 1%% — it divides by the plan's real size", got.Progress())
	}
}

// TestPlansOrderStartedFirst: a plan you are partway through is the one you came back for.
func TestPlansOrderStartedFirst(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// LeetCode 75 is the smaller plan, so size alone would put it second.
	if err := s.SetPlan(ctx, "leetcode-75", planOf("leetcode-75", "LeetCode 75",
		[]string{"two-sum"})); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}

	plans, err := s.Plans(ctx, "")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	if plans[0].Slug != "leetcode-75" {
		t.Errorf("first plan is %q, want leetcode-75 — started plans come first", plans[0].Slug)
	}
}

// TestUpsertPlansMerges guards the union in D-031: the registry is assembled from a seed
// list and a tag sweep, and a sweep that returns less than usual must not delete plans.
func TestUpsertPlansMerges(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)

	// A later sweep knows about only one of them, and reports no size for it.
	err := s.UpsertPlans(ctx, []leetcode.PlanBrief{
		{Slug: "leetcode-75", Name: "LeetCode 75", QuestionNum: 0},
	})
	if err != nil {
		t.Fatalf("UpsertPlans: %v", err)
	}

	plans, err := s.Plans(ctx, "")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("registry has %d plans after a partial sweep, want all 3 kept", len(plans))
	}
	for _, p := range plans {
		if p.Slug == "leetcode-75" && p.QuestionNum != 75 {
			t.Errorf("a zero size overwrote a known one: QuestionNum = %d, want 75", p.QuestionNum)
		}
	}
}

func TestPlanGroupsMapsChapters(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedPlans(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	plan := planOf("top-interview-150", "Top Interview 150",
		[]string{"two-sum"}, []string{"lru-cache"})
	if err := s.SetPlan(ctx, "top-interview-150", plan); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}

	groups, err := s.PlanGroups(ctx, "top-interview-150")
	if err != nil {
		t.Fatalf("PlanGroups: %v", err)
	}
	if groups["two-sum"] != "Array / String" {
		t.Errorf("two-sum is in %q, want Array / String", groups["two-sum"])
	}
	if groups["lru-cache"] != "Two Pointers" {
		t.Errorf("lru-cache is in %q, want Two Pointers", groups["lru-cache"])
	}
}
