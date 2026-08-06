package syncer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/grootbeat/leetui/internal/store"
)

func TestProblemsSyncPages(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	cl, _ := newFake(t, &fakeLeetCode{total: 250})
	sy := New(cl, st, 100)

	ch := make(chan Progress, 64)
	go func() { _ = sy.Problems(ctx, ch, true) }()
	updates := drainProgress(ch)

	n, err := st.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 250 {
		t.Fatalf("stored %d problems, want 250", n)
	}

	last := updates[len(updates)-1]
	if !last.Finished || last.Err != nil {
		t.Fatalf("final progress = %+v", last)
	}
	if last.Done != 250 || last.Total != 250 {
		t.Errorf("final progress = %d/%d, want 250/250", last.Done, last.Total)
	}

	// The cursor is cleared on success so the next sync starts fresh rather than
	// resuming from the end and appearing to do nothing.
	if v, _ := st.GetState(ctx, store.KeyProblemsCursor); store.Atoi(v) != 0 {
		t.Errorf("cursor = %q after a completed sync, want 0", v)
	}
	if v, _ := st.GetState(ctx, store.KeyProblemsSyncedAt); v == "" {
		t.Error("completed sync did not stamp problems_synced_at")
	}
}

// TestSyncResumesFromCheckpoint is the behaviour D-008 exists for: a cancelled sync must
// pick up where it stopped, not start over.
func TestSyncResumesFromCheckpoint(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	fake := &fakeLeetCode{total: 500}
	cl, _ := newFake(t, fake)
	sy := New(cl, st, 100)

	// Cancel after the second page lands.
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := make(chan Progress, 64)
	go func() { _ = sy.Problems(cancelCtx, ch, true) }()

	for p := range ch {
		if p.Done >= 200 && !p.Finished {
			cancel()
		}
	}

	mid, err := st.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mid == 0 || mid >= 500 {
		t.Fatalf("interrupted sync stored %d problems; expected a partial result", mid)
	}

	cursor, _ := st.GetState(ctx, store.KeyProblemsCursor)
	if store.Atoi(cursor) != mid {
		t.Fatalf("checkpoint = %s but %d rows are stored; they must agree", cursor, mid)
	}

	// Resume. The fake counts calls, so a restart-from-zero would be visible as extra
	// requests for pages already stored.
	callsBefore := fake.calls
	ch2 := make(chan Progress, 64)
	go func() { _ = sy.Problems(ctx, ch2, true) }()
	drainProgress(ch2)

	final, err := st.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if final != 500 {
		t.Fatalf("after resume: %d problems, want 500", final)
	}

	wantCalls := (500 - mid + 99) / 100
	if got := fake.calls - callsBefore; got > wantCalls+1 {
		t.Errorf("resume made %d requests, want about %d — it restarted instead of resuming",
			got, wantCalls)
	}
}

// TestSyncRetriesRateLimit: a rate-limited page must be retried, never skipped. Skipping
// would leave a silent hole in the local problem set.
func TestSyncRetriesRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("the retry path waits 30s")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	st := testStore(t)
	cl, _ := newFake(t, &fakeLeetCode{total: 300, rateAt: 2, rateOnce: true})
	sy := New(cl, st, 100)

	ch := make(chan Progress, 64)
	go func() { _ = sy.Problems(ctx, ch, true) }()
	updates := drainProgress(ch)

	var sawPause bool
	for _, p := range updates {
		if strings.Contains(p.Note, "rate limited") {
			sawPause = true
		}
	}
	if !sawPause {
		t.Error("a 429 did not surface as a visible pause")
	}

	// The context expires during the backoff, so this ends paused rather than complete.
	// What matters is that page 1 was kept and page 2 was not skipped past.
	//
	// Verify on a fresh context: ctx is deliberately expired by now.
	n, err := st.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Errorf("stored %d problems; want the 100 from the page before the rate limit", n)
	}
}

func TestSyncSurfacesErrors(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	cl, _ := newFake(t, &fakeLeetCode{total: 300, failAt: 2})
	sy := New(cl, st, 100)

	ch := make(chan Progress, 64)
	go func() { _ = sy.Problems(ctx, ch, true) }()
	updates := drainProgress(ch)

	last := updates[len(updates)-1]
	if last.Err == nil {
		t.Fatal("a server error did not reach the final progress message")
	}
	if !last.Finished {
		t.Error("the failing sync did not mark itself finished")
	}

	// The first page must survive so a resume does not redo it.
	if n, _ := st.Count(ctx); n != 100 {
		t.Errorf("stored %d problems, want the 100 fetched before the failure", n)
	}
}
