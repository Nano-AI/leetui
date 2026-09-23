package leetcode

import (
	"testing"
	"time"
)

// contestAt builds a 90-minute contest starting at a fixed instant.
func contestAt(start time.Time) ContestBrief {
	return ContestBrief{
		Title: "Weekly Contest 517", Slug: "weekly-contest-517",
		StartTime: start.Unix(), Duration: 5400,
	}
}

// TestPhaseBoundariesAreHalfOpen pins the two instants a contest changes state on.
//
// The start is INCLUSIVE and the end is EXCLUSIVE. Getting this backwards costs a user
// the first or last second of a contest, and the last second is the one people submit in.
func TestPhaseBoundariesAreHalfOpen(t *testing.T) {
	start := time.Date(2026, 8, 30, 2, 30, 0, 0, time.UTC)
	c := contestAt(start)
	end := start.Add(90 * time.Minute)

	cases := []struct {
		when time.Time
		want Phase
		why  string
	}{
		{start.Add(-time.Second), PhaseUpcoming, "one second before the start"},
		{start, PhaseLive, "the start instant itself is live, not upcoming"},
		{start.Add(89 * time.Minute), PhaseLive, "inside the contest"},
		{end.Add(-time.Second), PhaseLive, "the last second is still live"},
		{end, PhaseEnded, "the end instant is over, not live"},
		{end.Add(time.Hour), PhaseEnded, "long after"},
	}
	for _, tc := range cases {
		if got := c.PhaseAt(tc.when); got != tc.want {
			t.Errorf("PhaseAt(%s) = %v, want %v (%s)",
				tc.when.Format(time.TimeOnly), got, tc.want, tc.why)
		}
	}
}

// TestRemainingIsNeverNegative is what keeps the countdown from reading as a bug.
//
// A negative duration formats as "-1m23s". On a rail that is indistinguishable from a
// broken clock, and it is the state every ended contest would sit in.
func TestRemainingIsNeverNegative(t *testing.T) {
	start := time.Date(2026, 8, 30, 2, 30, 0, 0, time.UTC)
	c := contestAt(start)

	if d := c.Remaining(start.Add(3 * time.Hour)); d != 0 {
		t.Errorf("Remaining after the end = %v, want 0", d)
	}
	if d := c.Remaining(start.Add(-10 * time.Minute)); d != 10*time.Minute {
		t.Errorf("Remaining before the start = %v, want 10m — it counts to the START", d)
	}
	if d := c.Remaining(start.Add(10 * time.Minute)); d != 80*time.Minute {
		t.Errorf("Remaining while live = %v, want 80m — it counts to the END", d)
	}
}

// TestStartTimeIsSecondsNotMillis guards the unit the whole feature hangs on.
//
// LeetCode sends seconds here and milliseconds on other endpoints. Reading this field as
// milliseconds puts every contest in January 1970, and the countdown still "works".
func TestStartTimeIsSecondsNotMillis(t *testing.T) {
	// The real value observed for Weekly Contest 517.
	c := ContestBrief{StartTime: 1788057000, Duration: 5400}
	got := c.Start().UTC()
	if got.Year() != 2026 {
		t.Fatalf("Start() = %s, want a 2026 date; the field is Unix SECONDS", got)
	}
	if d := c.End().Sub(c.Start()); d != 90*time.Minute {
		t.Errorf("contest length = %v, want 90m; Duration is SECONDS", d)
	}
}

// TestSummaryLeavesDifficultyBlank is the guess this code deliberately does not make.
//
// Credit looks like a difficulty and is not one: a weekly's second problem is routinely
// an Easy and its third routinely a Hard. A blank renders as unknown, which is honest.
func TestSummaryLeavesDifficultyBlank(t *testing.T) {
	q := ContestQuestion{QuestionID: "4295", Title: "Count Indices", Slug: "count-indices", Credit: 6}
	s := q.Summary()
	if s.Difficulty != "" {
		t.Errorf("Summary().Difficulty = %q, want empty — credit is not a difficulty", s.Difficulty)
	}
	if s.Slug != "count-indices" || s.Title != "Count Indices" {
		t.Errorf("Summary() lost the identity: %+v", s)
	}
}

// TestSubmitContestNeedsAContest is the guard on the endpoint that actually scores.
//
// An empty contest slug would build "/contest/api//problems/x/submit/", which LeetCode
// answers for — with a submission that counts for nothing.
func TestSubmitContestNeedsAContest(t *testing.T) {
	c := New()
	_, err := c.SubmitContest(t.Context(), "", Submission{Slug: "two-sum", QuestionID: "1"})
	if err == nil {
		t.Fatal("SubmitContest with no contest slug succeeded; it must refuse")
	}
}
