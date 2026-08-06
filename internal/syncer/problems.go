package syncer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// Problems syncs the full problem list, resuming from the last checkpoint.
//
// It runs synchronously and emits progress on out, which it closes on return. Callers
// that need it in the background should run it in a goroutine and read out.
//
// Passing resume=false restarts from offset 0, which is what a manual "resync" does.
func (s *Syncer) Problems(ctx context.Context, out chan<- Progress, resume bool) error {
	defer close(out)

	skip := 0
	if resume {
		if v, err := s.store.GetState(ctx, store.KeyProblemsCursor); err == nil {
			skip = store.Atoi(v)
		}
	}

	total := 0
	if v, err := s.store.GetState(ctx, store.KeyProblemsTotal); err == nil {
		total = store.Atoi(v)
	}

	emit := func(p Progress) {
		p.Phase = PhaseProblems
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Done: skip, Total: total, Note: "starting"})

	for {
		if err := ctx.Err(); err != nil {
			// Cancellation is not a failure. The checkpoint is already written, so the
			// next run resumes from here.
			emit(Progress{Done: skip, Total: total, Note: "paused", Finished: true})
			return nil
		}

		page, pageTotal, err := s.client.ProblemPage(ctx, skip, s.pageSize)
		if err != nil {
			if errors.Is(err, leetcode.ErrRateLimited) {
				// Back off and retry the same page rather than skipping it — a skipped
				// page would leave a silent hole in the local problem set.
				emit(Progress{Done: skip, Total: total, Note: "rate limited, waiting 30s"})
				select {
				case <-time.After(30 * time.Second):
					continue
				case <-ctx.Done():
					emit(Progress{Done: skip, Total: total, Note: "paused", Finished: true})
					return nil
				}
			}
			emit(Progress{Done: skip, Total: total, Err: err, Finished: true})
			return fmt.Errorf("sync problems at offset %d: %w", skip, err)
		}

		if pageTotal > 0 && pageTotal != total {
			total = pageTotal
			_ = s.store.SetState(ctx, store.KeyProblemsTotal, store.Itoa(total))
		}

		if len(page) == 0 {
			break // reached the end
		}

		if err := s.store.UpsertSummaries(ctx, page); err != nil {
			emit(Progress{Done: skip, Total: total, Err: err, Finished: true})
			return err
		}

		skip += len(page)

		// Checkpoint AFTER the write commits, so a crash between the two re-fetches one
		// page rather than skipping it.
		if err := s.store.SetState(ctx, store.KeyProblemsCursor, store.Itoa(skip)); err != nil {
			emit(Progress{Done: skip, Total: total, Err: err, Finished: true})
			return err
		}

		emit(Progress{Done: skip, Total: total})

		if total > 0 && skip >= total {
			break
		}
	}

	// Completed: clear the cursor so the next sync starts fresh, and stamp the time.
	_ = s.store.SetState(ctx, store.KeyProblemsCursor, "0")
	_ = s.store.SetState(ctx, store.KeyProblemsSyncedAt, time.Now().Format(time.RFC3339))

	emit(Progress{Done: skip, Total: max(total, skip), Note: "done", Finished: true})
	return nil
}
