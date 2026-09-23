package leetcode

import "testing"

// TestPlanNormalizesEnums is the divergence that would otherwise be silent (D-031).
//
// studyPlanV2Detail answers "EASY" and "SOLVED" where every other endpoint answers "Easy"
// and "ac". Stored raw, every plan row would render as Easy — difficultyOf falls through
// to Easy for anything it does not recognise — and none of them would ever count as
// solved, so the picker's progress would read 0/150 forever.
func TestPlanNormalizesEnums(t *testing.T) {
	p := Plan{
		PlanBrief: PlanBrief{Slug: "top-interview-150", QuestionNum: 4},
		Groups: []PlanGroup{{
			Name: "Array / String",
			Questions: []PlanQuestion{
				{Slug: "a", Difficulty: "EASY", Status: "TO_DO"},
				{Slug: "b", Difficulty: "MEDIUM", Status: "SOLVED"},
				{Slug: "c", Difficulty: "HARD", Status: "ATTEMPTED"},
			},
		}},
	}
	p.normalize()

	got := p.Groups[0].Questions
	want := []struct {
		diff   Difficulty
		status Status
	}{
		{Easy, StatusNone},
		{Medium, StatusAccepted},
		{Hard, StatusAttempted},
	}
	for i, w := range want {
		if got[i].Difficulty != w.diff {
			t.Errorf("question %d difficulty = %q, want %q", i, got[i].Difficulty, w.diff)
		}
		if got[i].Status != w.status {
			t.Errorf("question %d status = %q, want %q", i, got[i].Status, w.status)
		}
	}
}

// TestPlanNormalizeKeepsUnknownValues: a new enum member must surface as something odd on
// screen, not be silently rewritten to Easy. Guessing would hide the schema change that
// the live test exists to catch.
func TestPlanNormalizeKeepsUnknownValues(t *testing.T) {
	p := Plan{Groups: []PlanGroup{{Questions: []PlanQuestion{
		{Slug: "a", Difficulty: "IMPOSSIBLE", Status: "PARTIAL"},
	}}}}
	p.normalize()

	q := p.Groups[0].Questions[0]
	if q.Difficulty != "IMPOSSIBLE" {
		t.Errorf("unknown difficulty became %q; it should have been left alone", q.Difficulty)
	}
	if q.Status != "PARTIAL" {
		t.Errorf("unknown status became %q; it should have been left alone", q.Status)
	}
}

// TestPlanQuestionsPreservesCurriculumOrder: the flattened order IS the plan, and it is
// what the store turns into plan_rank. Chapters must not interleave.
func TestPlanQuestionsPreservesCurriculumOrder(t *testing.T) {
	p := Plan{Groups: []PlanGroup{
		{Name: "Array / String", Questions: []PlanQuestion{{Slug: "a1"}, {Slug: "a2"}}},
		{Name: "Two Pointers", Questions: []PlanQuestion{{Slug: "t1"}}},
		{Name: "Sliding Window", Questions: []PlanQuestion{{Slug: "s1"}, {Slug: "s2"}}},
	}}

	var got []string
	for _, q := range p.Questions() {
		got = append(got, q.Slug)
	}
	want := []string{"a1", "a2", "t1", "s1", "s2"}
	if len(got) != len(want) {
		t.Fatalf("flattened to %d questions, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order is %v, want %v", got, want)
		}
	}

	groups := p.GroupOf()
	if groups["t1"] != "Two Pointers" {
		t.Errorf("t1 maps to %q, want Two Pointers", groups["t1"])
	}
	if groups["s2"] != "Sliding Window" {
		t.Errorf("s2 maps to %q, want Sliding Window", groups["s2"])
	}
}

// TestSeedPlansAreUnique guards the registry's dedupe input: a duplicated seed would cost
// an extra confirming request per launch for nothing.
func TestSeedPlansAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range SeedPlans() {
		if s == "" {
			t.Error("SeedPlans contains an empty slug")
		}
		if seen[s] {
			t.Errorf("SeedPlans lists %q twice", s)
		}
		seen[s] = true
	}
}
