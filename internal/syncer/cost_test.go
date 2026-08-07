package syncer_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
)

// What a full sync actually costs.
//
// Someone looking at the progress bar reasonably assumed it was heavy — it runs for a
// while and counts to four thousand. It is not: the loop streams one page at a time,
// upserts it, checkpoints, and discards it, so nothing ever holds the whole problem set.
//
// Measured 2026-08-07 on an M-series Mac at 8 req/s:
//
//	problems     4013
//	elapsed      19.9s
//	peak heap    4.7 MB
//	db on disk   3.0 MB
//
// The time is the RATE LIMITER, not work. At the shipped default of 2 req/s the same
// sync takes ~80s, and the machine is idle for almost all of it.
//
// LEETUI_SYNCBENCH=1 to run; it hits the live API.
func TestSyncCost(t *testing.T) {
	if os.Getenv("LEETUI_SYNCBENCH") == "" {
		t.Skip("set LEETUI_SYNCBENCH=1")
	}
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	creds, _ := auth.Load()
	cl := leetcode.New(leetcode.WithCredentials(creds), leetcode.WithRateLimit(8))
	sy := syncer.New(cl, st, 100)

	ch := make(chan syncer.Progress, 64)
	var peak uint64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ch {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > peak {
				peak = m.HeapAlloc
			}
		}
	}()

	start := time.Now()
	if err := sy.Problems(context.Background(), ch, false); err != nil {
		t.Fatalf("sync: %v", err)
	}
	elapsed := time.Since(start)
	<-done

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var dbBytes int64
	if fi, err := os.Stat(dir + "/leetui.db"); err == nil {
		dbBytes = fi.Size()
	}
	n, _ := st.Count(context.Background())

	fmt.Printf("\nproblems     %d\nelapsed      %s\npeak heap    %.1f MB\nafter sync   %.1f MB\ncumulative   %.1f MB (allocated over time, not resident)\ndb on disk   %.1f MB\ngoroutines   %d\n\n",
		n, elapsed.Round(time.Millisecond),
		float64(peak)/1e6, float64(m.HeapAlloc)/1e6, float64(m.TotalAlloc)/1e6,
		float64(dbBytes)/1e6, runtime.NumGoroutine())
}
