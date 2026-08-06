package syncer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/grootbeat/leetui/internal/leetcode"
	"github.com/grootbeat/leetui/internal/store"
)

// TestLiveSync talks to the real LeetCode API. It is opt-in: run with
//
//	LEETUI_LIVE=1 go test ./internal/syncer -run TestLiveSync -v
//
// It is the only check that our GraphQL documents still match LeetCode's schema, which
// is the thing most likely to break without any code change on our side.
func TestLiveSync(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st := testStore(t)
	cl := leetcode.New(leetcode.WithRateLimit(2))
	sy := New(cl, st, 50)

	ch := make(chan Progress, 32)
	go func() { _ = sy.Problems(ctx, ch, true) }()

	var last Progress
	for p := range ch {
		last = p
		if p.Done >= 100 {
			cancel() // 100 problems is enough to prove the schema
		}
	}

	// Verification runs on a fresh context: the sync context was cancelled above, and
	// reusing it would fail every query.
	check := context.Background()

	n, err := st.Count(check)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("synced %d problems (LeetCode reports %d total)", n, last.Total)

	if n < 50 {
		t.Fatalf("only %d problems synced", n)
	}
	if last.Total < 3000 {
		t.Errorf("LeetCode reported %d total problems, which looks wrong", last.Total)
	}

	rows, err := st.Query(check, store.Filter{Text: "two sum"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal(`searching "two sum" against live data returned nothing`)
	}
	t.Logf("search hit: %d. %s (%s, %.1f%%) tags=%v",
		rows[0].NumericID, rows[0].Title, rows[0].Difficulty, rows[0].AcRate, rows[0].Tags)
}

// TestLiveDetail fetches one real problem statement end to end: GraphQL -> store ->
// HTML conversion. Opt-in, same flag as TestLiveSync.
func TestLiveDetail(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st := testStore(t)
	cl := leetcode.New(leetcode.WithRateLimit(2))
	sy := New(cl, st, 50)

	// Seed the row the detail attaches to.
	if err := st.UpsertSummaries(ctx, []leetcode.ProblemSummary{
		{FrontendID: "1", Title: "Two Sum", Slug: "two-sum", Difficulty: leetcode.Easy},
	}); err != nil {
		t.Fatal(err)
	}

	d, err := sy.Detail(ctx, "two-sum", true)
	if err != nil {
		t.Fatalf("fetch detail: %v", err)
	}
	if !d.HasDetail || d.Content == "" {
		t.Fatal("detail came back empty")
	}
	if d.QuestionID == "" {
		t.Error("questionId is empty — the submit endpoint needs it")
	}
	if len(d.Snippets) == 0 {
		t.Error("no starter snippets")
	}
	if d.MetaData == "" {
		t.Error("metaData is empty — local codegen depends on it")
	}
	t.Logf("questionId=%s snippets=%d metaData=%s", d.QuestionID, len(d.Snippets), d.MetaData)
}
