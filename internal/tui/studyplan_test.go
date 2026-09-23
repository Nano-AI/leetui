package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// seedPlanRegistry puts a study plan registry in the model's store, standing in for what
// PlanRegistry would pull.
func seedPlanRegistry(t *testing.T, m Model) {
	t.Helper()
	err := m.store.UpsertPlans(context.Background(), []leetcode.PlanBrief{
		{Slug: "top-interview-150", Name: "Top Interview 150",
			Highlight: "Must-do List for Interview Prep", QuestionNum: 150},
		{Slug: "leetcode-75", Name: "LeetCode 75",
			Highlight: "Ace Coding Interview with 75 Qs", QuestionNum: 75},
		{Slug: "premium-algo-100", Name: "Premium Algo 100",
			QuestionNum: 100, PremiumOnly: true},
	})
	if err != nil {
		t.Fatalf("seed plan registry: %v", err)
	}
}

// seedPlanContents stores a two-chapter plan whose order disagrees with problem number,
// so any test that checks ordering is actually checking something.
func seedPlanContents(t *testing.T, m Model, slug string) {
	t.Helper()
	plan := leetcode.Plan{
		PlanBrief: leetcode.PlanBrief{Slug: slug, Name: slug, QuestionNum: 3},
		Groups: []leetcode.PlanGroup{
			{Name: "Array / String", Questions: []leetcode.PlanQuestion{
				{Slug: "lru-cache", Title: "LRU Cache", FrontendID: "146",
					Difficulty: leetcode.Medium},
				{Slug: "two-sum", Title: "Two Sum", FrontendID: "1",
					Difficulty: leetcode.Easy},
			}},
			{Name: "Two Pointers", Questions: []leetcode.PlanQuestion{
				{Slug: "trapping-rain-water", Title: "Trapping Rain Water",
					FrontendID: "42", Difficulty: leetcode.Hard},
			}},
		},
	}
	if err := m.store.SetPlan(context.Background(), slug, plan); err != nil {
		t.Fatalf("seed plan contents: %v", err)
	}
}

func TestPlanBrowserFiltersAsYouType(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)

	m = drive(t, m, key("P"))
	if m.mode != modePlan {
		t.Fatalf("P did not open the study plan browser; mode is %v", m.mode)
	}
	if len(m.plans) != 3 {
		t.Fatalf("loaded %d plans, want 3", len(m.plans))
	}

	m = drive(t, m, key("7"), key("5"))
	rows := m.visiblePlans()
	if len(rows) != 1 || rows[0].Slug != "leetcode-75" {
		t.Fatalf(`typing "75" left %+v, want just LeetCode 75`, rows)
	}

	out := m.View()
	if !strings.Contains(out, "LeetCode 75") {
		t.Error("the filtered plan is not on screen")
	}
	if strings.Contains(out, "Top Interview 150") {
		t.Error("a filtered-out plan is still on screen")
	}
}

// TestPlanPickIsOneStep is the difference from a company pack: there is no timeframe to
// ask about, so enter goes straight to the board rather than into a second picker.
func TestPlanPickIsOneStep(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)
	seedPlanContents(t, m, "top-interview-150")

	m = drive(t, m, key("P"))
	m = drive(t, m, key("enter"))

	if m.picking != pickNone {
		t.Fatalf("choosing a plan opened a picker (%v); a plan has no second question", m.picking)
	}
	if m.mode != modeBoard {
		t.Fatalf("choosing a plan left mode as %v, want the board", m.mode)
	}
	if !m.plan.Active() || m.plan.Slug != "top-interview-150" {
		t.Fatalf("plan is %+v, want top-interview-150 — the biggest plan sorts first", m.plan)
	}
}

func TestPlanFiltersAndOrdersTheBoard(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)
	seedPlanContents(t, m, "top-interview-150")

	m = drive(t, m, key("P"))
	m = drive(t, m, key("enter"))

	if len(m.rows) != 3 {
		t.Fatalf("board shows %d rows, want the 3 in the plan", len(m.rows))
	}
	// Curriculum order is what makes a plan more useful than a tag filter: LRU Cache is
	// problem 146 and comes first because the plan says so.
	want := []string{"lru-cache", "two-sum", "trapping-rain-water"}
	for i, w := range want {
		if m.rows[i].Slug != w {
			t.Errorf("row %d is %q, want %q — the board must follow plan order", i, m.rows[i].Slug, w)
		}
	}

	out := m.View()
	if !strings.Contains(out, "Top Interview 150") {
		t.Error("the active plan is not announced on screen")
	}

	// esc clears it and the whole problem set comes back.
	m = drive(t, m, key("esc"))
	if m.plan.Active() {
		t.Error("esc left the plan in place")
	}
	if len(m.rows) != 4 {
		t.Errorf("clearing the plan left %d rows, want all 4 seeded problems", len(m.rows))
	}
}

// TestPlanShowsChapterColumn covers the shared column slot: under a plan the last column
// reports chapters, and its header has to say so.
func TestPlanShowsChapterColumn(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)
	seedPlanContents(t, m, "top-interview-150")

	m = drive(t, m, key("P"))
	m = drive(t, m, key("enter"))

	if got := m.plan.Groups["two-sum"]; got != "Array / String" {
		t.Fatalf("two-sum's chapter is %q, want Array / String", got)
	}

	out := m.View()
	if !strings.Contains(strings.ToUpper(out), "CHAPTER") {
		t.Error("the last column is not headed CHAPTER while a plan is active")
	}
	if strings.Contains(strings.ToUpper(out), "ASKED BY") {
		t.Error("the board still says ASKED BY under a plan; the column reports chapters now")
	}
	if !strings.Contains(out, "Two Pointers") {
		t.Error("a row's chapter is not on screen")
	}
}

// TestPlanAndPackAreExclusive: both label the board and each sorts it differently, so the
// second one chosen must replace the first rather than sit alongside it.
func TestPlanAndPackAreExclusive(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedRegistry(t, m)
	seedPlanRegistry(t, m)
	seedPlanContents(t, m, "top-interview-150")

	err := m.store.SetPack(context.Background(), "google", "all", []leetcode.PackQuestion{
		{Slug: "two-sum", Title: "Two Sum", FrontendID: "1", Frequency: 0.9},
	})
	if err != nil {
		t.Fatalf("seed pack: %v", err)
	}

	// Pick a company pack first...
	m = drive(t, m, key("c"))
	m = drive(t, m, key("enter"))
	for i := 0; i < 4; i++ {
		m = drive(t, m, key("down"))
	}
	m = drive(t, m, key("enter"))
	if !m.pack.Active() {
		t.Fatalf("the pack did not apply: %+v", m.pack)
	}

	// ...then a study plan.
	m = drive(t, m, key("P"))
	m = drive(t, m, key("enter"))

	if !m.plan.Active() {
		t.Fatal("the plan did not apply")
	}
	if m.pack.Active() {
		t.Error("the pack survived; the board would be sorted by one and labelled with both")
	}
	if m.filter.Sort != "plan" || len(m.filter.Companies) != 0 {
		t.Errorf("filter is %+v, want plan-only", m.filter)
	}
}

// TestPlanPickerShowsProgress: the picker leads with how far through a plan you are,
// because that is the question the screen exists to answer. two-sum is accepted in the
// harness's seed data.
func TestPlanPickerShowsProgress(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)
	seedPlanContents(t, m, "leetcode-75")

	m = drive(t, m, key("P"))

	var got *int
	for _, p := range m.plans {
		if p.Slug == "leetcode-75" {
			solved := p.Solved
			got = &solved
		}
	}
	if got == nil {
		t.Fatal("leetcode-75 is missing from the registry")
	}
	if *got != 1 {
		t.Fatalf("Solved = %d, want 1 — only two-sum is accepted", *got)
	}

	if out := m.View(); !strings.Contains(out, "1/75") {
		t.Error("the picker does not show progress against the plan's real size")
	}
}

// TestPlanPickerNamesTheGate: a premium plan stays listed and says why, rather than
// vanishing from someone who may be paying for it.
func TestPlanPickerNamesTheGate(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedPlanRegistry(t, m)

	m = drive(t, m, key("P"))
	out := m.View()
	if !strings.Contains(out, "Premium Algo 100") {
		t.Error("the gated plan is hidden rather than listed")
	}
	if !strings.Contains(strings.ToLower(out), "premium") {
		t.Error("the gated plan does not name what is withholding it")
	}
}
