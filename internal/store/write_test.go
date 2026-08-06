package store

import (
	"context"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"testing"
)

func TestUpsertAndQuery(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	rows, err := s.Query(ctx, Filter{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	// Default sort is numeric ID, so 1, 42, 146 — not insertion order.
	if got := []int{rows[0].NumericID, rows[1].NumericID, rows[2].NumericID}; got[0] != 1 || got[1] != 42 || got[2] != 146 {
		t.Errorf("default sort = %v, want [1 42 146]", got)
	}
	if !rows[0].Solved() {
		t.Error("two-sum should read as solved")
	}
	if len(rows[0].Tags) != 2 {
		t.Errorf("two-sum tags = %v, want 2", rows[0].Tags)
	}
}

// TestUpsertIsIdempotent guards the sync path: a re-sync must update in place, never
// duplicate rows or FTS entries.
func TestUpsertIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	for i := 0; i < 3; i++ {
		if err := s.UpsertSummaries(ctx, sample()); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}

	n, err := s.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Fatalf("count = %d after 3 syncs, want 3", n)
	}

	// FTS must not accumulate duplicates either, or search results repeat.
	rows, err := s.Query(ctx, Filter{Text: "sum"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("search 'sum' returned %d rows after 3 syncs, want 1", len(rows))
	}
}

// TestSetStatusOnlyMovesForward: LeetCode keeps your best result, and so does this. A
// wrong answer on a problem you already solved must not downgrade the row to TRIED.
func TestSetStatusOnlyMovesForward(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	statusOf := func(slug string) string {
		t.Helper()
		d, err := s.Get(ctx, slug)
		if err != nil {
			t.Fatalf("get %s: %v", slug, err)
		}
		return d.Status
	}

	// lru-cache starts untouched.
	if got := statusOf("lru-cache"); got != "" {
		t.Fatalf("lru-cache starts as %q, want empty", got)
	}

	if err := s.SetStatus(ctx, "lru-cache", leetcode.StatusAttempted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got := statusOf("lru-cache"); got != "notac" {
		t.Errorf("after a wrong answer: %q, want notac", got)
	}

	if err := s.SetStatus(ctx, "lru-cache", leetcode.StatusAccepted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got := statusOf("lru-cache"); got != "ac" {
		t.Errorf("after Accepted: %q, want ac", got)
	}

	// The regression this guards: a later wrong answer must leave it solved.
	if err := s.SetStatus(ctx, "lru-cache", leetcode.StatusAttempted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got := statusOf("lru-cache"); got != "ac" {
		t.Errorf("a wrong answer downgraded a solved problem to %q", got)
	}
}
