package store

import (
	"context"
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
