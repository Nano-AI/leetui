package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// seedContestSchedule puts a schedule in the model's store, standing in for what
// ContestSchedule would pull. The three rows are deliberately one per phase, so a test
// that checks how a phase renders is actually checking something.
func seedContestSchedule(t *testing.T, m Model) {
	t.Helper()
	now := time.Now()
	err := m.store.UpsertContests(context.Background(), []leetcode.ContestBrief{
		{Title: "Weekly Contest 900", Slug: "weekly-900",
			StartTime: now.Add(2 * time.Hour).Unix(), Duration: 5400},
		{Title: "Weekly Contest 899", Slug: "weekly-899",
			StartTime: now.Add(-30 * time.Minute).Unix(), Duration: 5400},
		{Title: "Weekly Contest 898", Slug: "weekly-898",
			StartTime: now.Add(-8 * 24 * time.Hour).Unix(), Duration: 5400},
	})
	if err != nil {
		t.Fatalf("seed contest schedule: %v", err)
	}
}

// seedContestContents stores a contest whose credit order disagrees with problem number,
// so an ordering check is meaningful.
func seedContestContents(t *testing.T, m Model, slug string) {
	t.Helper()
	c := &leetcode.Contest{
		ContestBrief: leetcode.ContestBrief{
			Title: slug, Slug: slug,
			StartTime: time.Now().Add(-30 * time.Minute).Unix(), Duration: 5400,
		},
		Questions: []leetcode.ContestQuestion{
			{Slug: "trapping-rain-water", Title: "Trapping Rain Water", QuestionID: "42", Credit: 6},
			{Slug: "two-sum", Title: "Two Sum", QuestionID: "1", Credit: 3},
		},
	}
	if err := m.store.SetContest(context.Background(), c); err != nil {
		t.Fatalf("seed contest contents: %v", err)
	}
}

// TestContestBrowserOpensAndCloses is the mode's front door: C in, esc out.
func TestContestBrowserOpensAndCloses(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedContestSchedule(t, m)

	m = drive(t, m, key("C"))
	if m.mode != modeContest {
		t.Fatalf("mode after C = %v, want modeContest", m.mode)
	}

	m = drive(t, m, key("esc"))
	if m.mode != modeBoard {
		t.Errorf("mode after esc = %v, want modeBoard", m.mode)
	}
}

// TestContestBrowserListsTheSchedule checks the registry reaches the screen, and that a
// live contest is the row that says how long is left.
func TestContestBrowserListsTheSchedule(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedContestSchedule(t, m)

	m = drive(t, m, key("C"))
	m = drive(t, m, m.loadContests()())

	out := stripANSI(m.View())
	for _, want := range []string{"Weekly Contest 900", "Weekly Contest 899"} {
		if !strings.Contains(out, want) {
			t.Errorf("contest browser does not list %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "left") {
		t.Errorf("no countdown on the live contest; the phase is the point of this screen:\n%s", out)
	}
}

// TestContestFiltersAndSortsTheBoard is what picking a contest is for: the board narrows
// to its problems and follows its running order.
func TestContestFiltersAndSortsTheBoard(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedContestSchedule(t, m)
	seedContestContents(t, m, "weekly-899")

	m = drive(t, m, key("C"))
	m = drive(t, m, m.loadContests()())

	// weekly-900 is newest and sorts first; the live one is next.
	m = drive(t, m, key("down"), key("enter"))

	if m.mode != modeBoard {
		t.Fatalf("mode after enter = %v, want modeBoard", m.mode)
	}
	if m.filter.Contest != "weekly-899" {
		t.Fatalf("filter.Contest = %q, want weekly-899", m.filter.Contest)
	}
	if m.filter.Sort != "contest" {
		t.Errorf("sort = %q, want contest — credit order is the running order", m.filter.Sort)
	}
	if !m.contest.Active() {
		t.Error("contest selection is not active after picking one")
	}
}

// TestContestClockCountsDownAndStops pins the three phases the rail has to word.
//
// The ended case returning empty is the one that matters: a countdown that has run out is
// not information, and "0s left" on the rail reads as a stuck clock.
func TestContestClockCountsDownAndStops(t *testing.T) {
	start := time.Date(2026, 8, 30, 2, 30, 0, 0, time.UTC)
	m := Model{contest: contestSel{
		Slug: "weekly-517", Title: "Weekly Contest 517",
		StartTime: start.Unix(), Duration: 5400,
	}}

	if got := m.contestClock(start.Add(-10 * time.Minute)); !strings.Contains(got, "starts in") {
		t.Errorf("clock before the start = %q, want a 'starts in' phrase", got)
	}
	if got := m.contestClock(start.Add(10 * time.Minute)); !strings.Contains(got, "left") {
		t.Errorf("clock while live = %q, want a 'left' phrase", got)
	}
	if got := m.contestClock(start.Add(3 * time.Hour)); got != "" {
		t.Errorf("clock after the end = %q, want empty", got)
	}

	if !m.contestLive(start.Add(10 * time.Minute)) {
		t.Error("contestLive is false during the contest; the rail colours the clock on it")
	}
	if m.contestLive(start.Add(3 * time.Hour)) {
		t.Error("contestLive is true after the contest ended")
	}
}

// TestContestAutoPullsWhenTheClockRunsOut is the transition the feature is pointed at.
//
// You pick the contest before it starts, because there is nothing else to do while
// waiting, and the questions do not exist until the clock reaches zero. If the tick does
// not notice the crossing, the countdown hits 00:00 and the board sits empty until you
// think to press C and enter again — in the worst minute to be navigating menus.
func TestContestAutoPullsWhenTheClockRunsOut(t *testing.T) {
	m := boot(t, true, 100, 30)

	// A contest that started one second ago, picked while it was still upcoming.
	m.contest = contestSel{
		Slug: "weekly-899", Title: "Weekly Contest 899",
		StartTime: time.Now().Add(-time.Second).Unix(), Duration: 5400,
	}
	m.contestPhase = leetcode.PhaseUpcoming

	if !m.contestOpened(time.Now()) {
		t.Fatal("the upcoming→live crossing was not noticed; the board would never refill")
	}

	// It must fire ONCE. After the phase is recorded, a later tick is not another pull.
	m.contestPhase = leetcode.PhaseLive
	if m.contestOpened(time.Now()) {
		t.Error("crossing reported again while already live; this would refetch every second")
	}
}

// TestContestDoesNotAutoPullBeforeTheStart is the other half: a contest still counting
// down must not be pulled on every tick.
func TestContestDoesNotAutoPullBeforeTheStart(t *testing.T) {
	m := boot(t, true, 100, 30)
	m.contest = contestSel{
		Slug: "weekly-900", Title: "Weekly Contest 900",
		StartTime: time.Now().Add(time.Hour).Unix(), Duration: 5400,
	}
	m.contestPhase = leetcode.PhaseUpcoming

	if m.contestOpened(time.Now()) {
		t.Error("a contest an hour away reported as just-opened")
	}
}

// TestContestIsExclusiveWithAPlan keeps the board from being sorted by one curated list
// and labelled with another.
func TestContestIsExclusiveWithAPlan(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedPlanRegistry(t, m)
	seedContestSchedule(t, m)

	m = drive(t, m, key("P"))
	m = drive(t, m, m.loadPlans()())
	m = drive(t, m, key("enter"))
	if !m.plan.Active() {
		t.Fatal("no plan active after picking one")
	}

	m = drive(t, m, key("C"))
	m = drive(t, m, m.loadContests()())
	m = drive(t, m, key("enter"))

	if m.plan.Active() {
		t.Error("a plan is still active after picking a contest; the board would be sorted by one and labelled with the other")
	}
	if m.filter.Plan != "" {
		t.Errorf("filter.Plan = %q, want empty after a contest was picked", m.filter.Plan)
	}
}

// TestContestBrowserFiltersAsYouType is the habit the other two list modes already have.
func TestContestBrowserFiltersAsYouType(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedContestSchedule(t, m)

	m = drive(t, m, key("C"))
	m = drive(t, m, m.loadContests()())
	if len(m.visibleContests()) != 3 {
		t.Fatalf("got %d contests before typing, want 3", len(m.visibleContests()))
	}

	m = drive(t, m, key("9"), key("0"), key("0"))
	rows := m.visibleContests()
	if len(rows) != 1 || rows[0].Slug != "weekly-900" {
		t.Fatalf("typing 900 gave %d rows (%+v), want just weekly-900", len(rows), rows)
	}
}
