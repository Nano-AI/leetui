package store

import (
	"context"
	"testing"
	"time"
)

func TestTodoRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if got, _ := s.IsTodo(ctx, "two-sum"); got {
		t.Fatal("an empty list reported a member")
	}

	if err := s.AddTodo(ctx, "two-sum", "warmup"); err != nil {
		t.Fatalf("AddTodo: %v", err)
	}
	if got, _ := s.IsTodo(ctx, "two-sum"); !got {
		t.Error("the added problem is not on the list")
	}

	entries, err := s.Todos(ctx)
	if err != nil {
		t.Fatalf("Todos: %v", err)
	}
	if len(entries) != 1 || entries[0].Note != "warmup" {
		t.Fatalf("list is %+v", entries)
	}

	if err := s.RemoveTodo(ctx, "two-sum"); err != nil {
		t.Fatalf("RemoveTodo: %v", err)
	}
	if got, _ := s.IsTodo(ctx, "two-sum"); got {
		t.Error("the removed problem is still listed")
	}
}

// TestReaddDoesNotResetPosition: the list is a queue, and correcting a note must not send
// the oldest item to the back of it.
func TestReaddDoesNotResetPosition(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.AddTodo(ctx, "two-sum", ""); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Todos(ctx)
	firstAdded := before[0].AddedAt

	time.Sleep(1100 * time.Millisecond) // added_at has one-second resolution
	if err := s.AddTodo(ctx, "lru-cache", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTodo(ctx, "two-sum", "a better note"); err != nil {
		t.Fatal(err)
	}

	after, err := s.Todos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after[0].Slug != "two-sum" {
		t.Errorf("re-adding moved two-sum to position of %q; the list is a queue", after[0].Slug)
	}
	if !after[0].AddedAt.Equal(firstAdded) {
		t.Errorf("added_at moved from %v to %v", firstAdded, after[0].AddedAt)
	}
	if after[0].Note != "a better note" {
		t.Errorf("the note did not update: %q", after[0].Note)
	}
}

// TestTodoSurvivesAResync is the reason it is a separate table: the problems table is a
// cache of LeetCode's data and a sync rewrites it, but a curated list is the user's.
func TestTodoSurvivesAResync(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTodo(ctx, "two-sum", "keep me"); err != nil {
		t.Fatal(err)
	}

	// A full re-sync, exactly as the syncer performs it.
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	if got, _ := s.IsTodo(ctx, "two-sum"); !got {
		t.Error("a re-sync erased the todo list")
	}
}

func TestTodoFilterAndOrder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	// lru-cache added first, so it leads the queue despite the higher number.
	if err := s.AddTodo(ctx, "lru-cache", ""); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err := s.AddTodo(ctx, "two-sum", ""); err != nil {
		t.Fatal(err)
	}

	rows, err := s.Query(ctx, Filter{TodoOnly: true, Sort: "todo"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("todo filter returned %d rows, want 2", len(rows))
	}
	if rows[0].Slug != "lru-cache" {
		t.Errorf("queue order is wrong: first is %q, want lru-cache (added first)", rows[0].Slug)
	}

	// And an unfiltered query still returns everything.
	all, err := s.Query(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("unfiltered query returned %d rows, want 3", len(all))
	}
}

// TestTodoAcceptsAnUnsyncedProblem: an agent may queue something before this machine has
// ever synced it, and the entry has to survive until it does.
func TestTodoAcceptsAnUnsyncedProblem(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.AddTodo(ctx, "a-problem-not-synced-here", "from a job description"); err != nil {
		t.Fatalf("AddTodo for an unknown problem: %v", err)
	}
	entries, err := s.Todos(ctx)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %+v, err = %v", entries, err)
	}
}
