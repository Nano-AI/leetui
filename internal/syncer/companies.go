package syncer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Company packs (D-006)
// ---------------------------------------------------------------------------
//
// Two jobs, deliberately separate:
//
//	CompanyRegistry  one request, works signed out, fills the browse list
//	Pack             one company + timeframe, on demand, premium
//
// Packs are never synced in bulk. All 984 companies across five timeframes would be
// roughly 5,000 requests at the shared rate limit — forty minutes of traffic to answer a
// question the user asked about one company.

// CompanyRegistry refreshes the list of companies.
//
// It needs no session, so it runs on first launch and gives a signed-out user something
// real to browse. It emits on out and closes it, like every other sync job.
func (s *Syncer) CompanyRegistry(ctx context.Context, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhaseCompanies
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Note: "companies"})

	companies, err := s.client.CompanyTags(ctx)
	if err != nil {
		emit(Progress{Err: err, Finished: true})
		return fmt.Errorf("sync company registry: %w", err)
	}
	if err := s.store.UpsertCompanies(ctx, companies); err != nil {
		emit(Progress{Err: err, Finished: true})
		return err
	}

	_ = s.store.SetState(ctx, store.KeyCompaniesSyncedAt, time.Now().Format(time.RFC3339))
	emit(Progress{Done: len(companies), Total: len(companies), Note: "done", Finished: true})
	return nil
}

// Pack pulls one company's problem list for one timeframe.
//
// It replaces that slice wholesale rather than resuming from an offset. A pack is
// hundreds of rows, not thousands of pages, and a half-written pack that looked complete
// would quietly misrepresent what a company asks. The store's write is one transaction,
// so a cancelled pack leaves the previous one intact.
func (s *Syncer) Pack(ctx context.Context, company string, tf leetcode.Timeframe, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhaseCompanies
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	if !tf.Valid() {
		err := fmt.Errorf("unknown timeframe %q", tf)
		emit(Progress{Err: err, Finished: true})
		return err
	}

	// PackSize both sizes the progress bar and proves the pack exists: LeetCode answers
	// an unknown company with null, which paging alone would report as an empty list.
	total, err := s.client.PackSize(ctx, company, tf)
	if err != nil {
		emit(Progress{Err: err, Finished: true})
		return fmt.Errorf("size pack %s/%s: %w", company, tf, err)
	}
	emit(Progress{Total: total, Note: company})

	var all []leetcode.PackQuestion
	for skip := 0; skip < total; {
		if err := ctx.Err(); err != nil {
			emit(Progress{Done: len(all), Total: total, Note: "cancelled", Finished: true})
			return nil
		}

		page, err := s.client.CompanyPage(ctx, company, tf, skip, s.pageSize)
		if err != nil {
			if errors.Is(err, leetcode.ErrRateLimited) {
				emit(Progress{Done: len(all), Total: total, Note: "rate limited, waiting 30s"})
				select {
				case <-time.After(30 * time.Second):
					continue
				case <-ctx.Done():
					emit(Progress{Done: len(all), Total: total, Note: "cancelled", Finished: true})
					return nil
				}
			}
			emit(Progress{Done: len(all), Total: total, Err: err, Finished: true})
			return fmt.Errorf("sync pack %s/%s at %d: %w", company, tf, skip, err)
		}

		if len(page.Questions) == 0 {
			break // the pack is shorter than its reported size; take what arrived
		}
		all = append(all, page.Questions...)
		skip += len(page.Questions)
		emit(Progress{Done: len(all), Total: total, Note: company})

		if !page.HasMore {
			break
		}
	}

	if err := s.store.SetPack(ctx, company, string(tf), all); err != nil {
		emit(Progress{Done: len(all), Total: total, Err: err, Finished: true})
		return err
	}
	_ = s.store.SetState(ctx, store.PackKey(company, string(tf)), time.Now().Format(time.RFC3339))

	emit(Progress{Done: len(all), Total: total, Note: "done", Finished: true})
	return nil
}
