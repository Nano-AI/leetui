package syncer

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// TestLiveStudyPlanSchema checks the study plan documents against the real endpoint.
// Opt in with:
//
//	LEETUI_LIVE=1 go test ./internal/syncer -run TestLiveStudyPlan -v
//
// It runs SIGNED OUT, and unlike the premium suite that is not a compromise: plan contents
// are genuinely free, so this exercises the real path a free account takes (D-031).
func TestLiveStudyPlanSchema(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cl := leetcode.New(leetcode.WithRateLimit(2))

	// A plan arrives whole. If this ever needs paging, the sync's one-request assumption
	// is wrong and the count below is what says so.
	plan, err := cl.StudyPlan(ctx, "top-interview-150")
	if err != nil {
		t.Fatalf("StudyPlan(top-interview-150): %v", err)
	}
	got := len(plan.Questions())
	if got != plan.QuestionNum {
		t.Errorf("got %d questions but the plan claims %d — a page may have been dropped",
			got, plan.QuestionNum)
	}
	if len(plan.Groups) < 2 {
		t.Errorf("got %d chapters, expected the plan to be split into many", len(plan.Groups))
	}
	if plan.Name == "" || plan.Highlight == "" {
		t.Errorf("plan metadata is thin: %+v", plan.Brief())
	}
	t.Logf("top-interview-150: %d questions across %d chapters", got, len(plan.Groups))

	// The enums this endpoint uses are its own, and normalising them is what keeps the
	// board honest. Anything outside leetui's vocabulary here means the mapping is stale.
	for _, q := range plan.Questions() {
		switch q.Difficulty {
		case leetcode.Easy, leetcode.Medium, leetcode.Hard:
		default:
			t.Fatalf("%s has difficulty %q after normalising — the enum has moved",
				q.Slug, q.Difficulty)
		}
		switch q.Status {
		case leetcode.StatusNone, leetcode.StatusAccepted, leetcode.StatusAttempted:
		default:
			t.Fatalf("%s has status %q after normalising — the enum has moved", q.Slug, q.Status)
		}
	}

	// A premium plan must come back as a gate carrying its public fields, not as a bare
	// error — that is what the picker's PREMIUM note renders from.
	locked, err := cl.StudyPlan(ctx, "premium-algo-100")
	if !errors.Is(err, leetcode.ErrPremiumRequired) {
		t.Errorf("StudyPlan(premium-algo-100) signed out returned %v, want ErrPremiumRequired", err)
	}
	if locked.Name == "" || locked.QuestionNum == 0 {
		t.Errorf("a gated plan came back with nothing to show: %+v", locked.Brief())
	}

	// A slug LeetCode does not have must be an error, not an empty plan.
	if _, err := cl.StudyPlan(ctx, "definitely-not-a-study-plan"); !errors.Is(err, leetcode.ErrNotFound) {
		t.Errorf("StudyPlan on a bogus slug returned %v, want ErrNotFound", err)
	}

	// The tag sweep has to keep finding something, or discovery has quietly become a
	// seed-list-only registry.
	var found int
	for _, tag := range leetcode.DiscoveryTags() {
		plans, err := cl.StudyPlansByTag(ctx, tag)
		if err != nil {
			t.Errorf("StudyPlansByTag(%s): %v", tag, err)
			continue
		}
		t.Logf("tag %-22s %d plans", tag, len(plans))
		found += len(plans)
	}
	if found == 0 {
		t.Error("no tag returned any plan — the tag vocabulary has changed")
	}
}

// TestLiveStudyPlanEndToEnd pulls a real plan through the syncer into a real database and
// asks the board for it, which is what pressing P and choosing a plan actually does.
//
// The schema test proves the wire format is understood. This proves the parts agree:
// normalisation survives the round trip, rank comes back as curriculum order, and the
// chapter map lines up with the rows.
func TestLiveStudyPlanEndToEnd(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st := testStore(t)
	sy := New(leetcode.New(leetcode.WithRateLimit(2)), st, 100)

	ch := make(chan Progress, 32)
	go func() { _ = sy.StudyPlan(ctx, "leetcode-75", ch) }()
	progress := drainProgress(ch)
	if last := progress[len(progress)-1]; last.Err != nil {
		t.Fatalf("sync failed: %v", last.Err)
	}

	rows, err := st.Query(ctx, store.Filter{Plan: "leetcode-75", Sort: "plan"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) < 70 {
		t.Fatalf("board shows %d rows for LeetCode 75, want ~75", len(rows))
	}

	// Every row must carry a real difficulty, or normalisation did not survive the store.
	for _, r := range rows {
		switch r.Difficulty {
		case "Easy", "Medium", "Hard":
		default:
			t.Fatalf("%s stored difficulty %q — normalisation was lost on the way in",
				r.Slug, r.Difficulty)
		}
	}

	// Plan order must not be ID order. LeetCode 75 opens on Two Pointers, not on problem 1.
	ascending := true
	for i := 1; i < len(rows); i++ {
		if rows[i].NumericID < rows[i-1].NumericID {
			ascending = false
			break
		}
	}
	if ascending {
		t.Error("the board came back in ID order — plan_rank is not being applied")
	}

	groups, err := st.PlanGroups(ctx, "leetcode-75")
	if err != nil {
		t.Fatalf("PlanGroups: %v", err)
	}
	for _, r := range rows {
		if groups[r.Slug] == "" {
			t.Fatalf("%s has no chapter; the CHAPTER column would be blank for it", r.Slug)
		}
	}
	t.Logf("LeetCode 75: %d rows, first is %d %q in %q",
		len(rows), rows[0].NumericID, rows[0].Title, groups[rows[0].Slug])
}

// TestLiveStudyPlanSlugs confirms every shipped seed slug still resolves.
//
// This is the test that catches a rename. "sql-50" was already wrong once — the website's
// URL says sql-50 and the plan slug is top-sql-50 — and a silently missing plan is
// indistinguishable from one the user simply has not scrolled to.
func TestLiveStudyPlanSlugs(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cl := leetcode.New(leetcode.WithRateLimit(2))

	for _, slug := range leetcode.SeedPlans() {
		plan, err := cl.StudyPlan(ctx, slug)
		switch {
		case err == nil:
			t.Logf("%-34s %-36s %d questions", slug, plan.Name, plan.QuestionNum)
		case errors.Is(err, leetcode.ErrPremiumRequired):
			// Expected signed out for the gated ones, and still proof the slug is real.
			t.Logf("%-34s %-36s %d questions (premium)", slug, plan.Name, plan.QuestionNum)
		case errors.Is(err, leetcode.ErrNotFound):
			t.Errorf("seed plan %q no longer exists — rename or drop it in SeedPlans", slug)
		default:
			t.Errorf("seed plan %q: %v", slug, err)
		}
	}
}
