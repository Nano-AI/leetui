package store

import (
	"context"
	"errors"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func TestEditorialRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := s.GetEditorial(ctx, "two-sum"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty cache returned %v, want ErrNotFound", err)
	}

	sol := &leetcode.Solution{
		ID: "7", Title: "Two Sum", Content: "## Approach 1\n\nUse a hash table.",
		CanSeeDetail: true, HasVideoSolution: true,
	}
	if err := s.SetEditorial(ctx, "two-sum", sol); err != nil {
		t.Fatalf("SetEditorial: %v", err)
	}

	got, err := s.GetEditorial(ctx, "two-sum")
	if err != nil {
		t.Fatalf("GetEditorial: %v", err)
	}
	if got.Content != sol.Content || !got.CanSee || !got.HasVideo {
		t.Errorf("round trip lost data: %+v", got)
	}
}

// TestEditorialCachesTheLock is the whole reason gated editorials are stored at all:
// without the row, every keypress would re-request a gate that cannot open.
func TestEditorialCachesTheLock(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	locked := &leetcode.Solution{
		ID: "554", Title: "Meeting Rooms II", Content: "",
		PaidOnly: true, CanSeeDetail: false, HasVideoSolution: true,
	}
	if err := s.SetEditorial(ctx, "lru-cache", locked); err != nil {
		t.Fatalf("SetEditorial: %v", err)
	}

	got, err := s.GetEditorial(ctx, "lru-cache")
	if err != nil {
		t.Fatalf("GetEditorial: %v", err)
	}
	if got.CanSee {
		t.Error("a gated editorial came back readable")
	}
	// The title and the video flag are public even when the body is not, and the pane
	// uses both to say what is behind the lock.
	if got.Title != "Meeting Rooms II" || !got.HasVideo {
		t.Errorf("lost the public fields of a locked editorial: %+v", got)
	}
}
