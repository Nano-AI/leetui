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
// Contests (D-036)
// ---------------------------------------------------------------------------
//
// Two jobs, mirroring the plan pair:
//
//	ContestSchedule  the upcoming contests — one request, signed out
//	Contest          one contest's questions — two requests, signed out
//
// Both are cheap enough to run on a keypress. The reason that matters more here than
// anywhere else in this package: during a live contest the questions become readable AT
// the start and not a moment before, so refreshing is the normal way to use this, not a
// recovery from failure. A job that cost twenty requests could not be pressed repeatedly.

// ContestSchedule refreshes the list of upcoming contests.
//
// One request, signed out. Merges into whatever the table already knows, so past
// contests already pulled survive a schedule refresh that no longer mentions them.
func (s *Syncer) ContestSchedule(ctx context.Context, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhaseContests
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Note: "schedule"})

	list, err := s.client.UpcomingContests(ctx)
	if err != nil {
		emit(Progress{Err: err, Finished: true})
		return fmt.Errorf("sync contest schedule: %w", err)
	}
	if err := s.store.UpsertContests(ctx, list); err != nil {
		emit(Progress{Err: err, Finished: true})
		return err
	}
	if err := s.store.SetState(ctx, store.KeyContestsSyncedAt, time.Now().Format(time.RFC3339)); err != nil {
		emit(Progress{Done: len(list), Total: len(list), Err: err, Finished: true})
		return err
	}

	emit(Progress{Done: len(list), Total: len(list), Note: "done", Finished: true})
	return nil
}

// ContestDetail fetches one contest problem's statement and stores it.
//
// The contest-aware sibling of Detail, and it exists because Detail cannot do this job
// while a contest is live: the ordinary question query returns null for a problem that is
// not yet in the problem set. Falls back to Detail once the contest is over and the
// problem has joined the problem set, so upsolving takes the ordinary path.
//
// `force` is ignored deliberately while the problem has no stored detail: there is no
// cached answer to prefer, and during a contest the fetch is the only source there is.
func (s *Syncer) ContestDetail(ctx context.Context, contest, slug string) (*store.Detail, error) {
	if d, err := s.store.Get(ctx, slug); err == nil && d.HasDetail {
		return d, nil
	}

	p, err := s.client.ContestProblem(ctx, contest, slug)
	if err != nil {
		// Not in the contest's own view either. Once a contest ends its problems become
		// ordinary ones, so the plain path is the right second guess rather than a
		// failure.
		if d, derr := s.Detail(ctx, slug, false); derr == nil {
			return d, nil
		}
		return nil, err
	}

	if err := s.store.SetDetail(ctx, p); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, slug)
}

// Contest pulls one contest's questions.
//
// Two requests and no paging loop: a contest is four problems, or five for a biweekly.
//
// AN EMPTY QUESTION LIST IS NOT AN ERROR. Before the start LeetCode returns an empty list
// rather than saying "not yet", so this reports how many arrived and lets the caller
// decide what that means — the phase is what tells "not open yet" apart from "no such
// contest", and the syncer does not own the clock.
func (s *Syncer) Contest(ctx context.Context, slug string, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhaseContests
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Note: slug})

	for {
		c, err := s.client.ContestDetail(ctx, slug)
		if errors.Is(err, leetcode.ErrRateLimited) {
			emit(Progress{Note: "rate limited, waiting 30s"})
			select {
			case <-time.After(30 * time.Second):
				continue
			case <-ctx.Done():
				emit(Progress{Note: "cancelled", Finished: true})
				return nil
			}
		}
		if err != nil {
			emit(Progress{Err: err, Finished: true})
			return fmt.Errorf("sync contest %s: %w", slug, err)
		}

		n := len(c.Questions)
		if err := s.store.SetContest(ctx, c); err != nil {
			emit(Progress{Err: err, Finished: true})
			return err
		}
		if err := s.store.SetState(ctx, store.ContestKey(slug), time.Now().Format(time.RFC3339)); err != nil {
			emit(Progress{Done: n, Total: n, Err: err, Finished: true})
			return err
		}

		emit(Progress{Done: n, Total: n, Note: "done", Finished: true})
		return nil
	}
}
