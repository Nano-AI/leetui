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
// Study plans (D-031)
// ---------------------------------------------------------------------------
//
// Two jobs, mirroring the company pair:
//
//	PlanRegistry  the list of plans — a seed list unioned with a tag sweep, signed out
//	StudyPlan     one plan's problems — ONE request, signed out
//
// Where a company pack is 24 requests and premium, a plan is one request and free. That
// is why the registry can afford to confirm every seed slug with a real fetch while a
// company registry has to take LeetCode's word for its 984 entries.

// PlanRegistry refreshes the list of study plans.
//
// It works signed out and runs on first press, so a free account has something real to
// browse immediately.
//
// The registry is a UNION of two incomplete sources, and it has to be:
//
//	tag sweep   finds plans added after this release, but LeetCode's tag vocabulary is a
//	            short curated set that omits Top 100 Liked, Binary Search, Graph Theory
//	            and every premium plan
//	seed list   covers those, but cannot know about a plan that did not exist at release
//
// Neither alone is the catalogue. A seed slug that has been retired simply fails its
// detail fetch and is skipped rather than failing the job — one dead plan must not cost
// the user the other sixteen.
func (s *Syncer) PlanRegistry(ctx context.Context, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhasePlans
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Note: "study plans"})

	// Slug order is preserved so the seeds are confirmed first and the picker fills from
	// the top. seen also dedupes the tag sweep against itself: one plan carries several
	// tags, and Top Interview 150 appears under three of them.
	seen := map[string]bool{}
	var order []string
	briefs := map[string]leetcode.PlanBrief{}

	for _, slug := range leetcode.SeedPlans() {
		if !seen[slug] {
			seen[slug] = true
			order = append(order, slug)
		}
	}

	tags := leetcode.DiscoveryTags()
	for _, tag := range tags {
		if err := ctx.Err(); err != nil {
			emit(Progress{Note: "cancelled", Finished: true})
			return nil
		}
		found, err := s.client.StudyPlansByTag(ctx, tag)
		if err != nil {
			// A tag sweep is an enhancement over the seed list, never a prerequisite.
			// Losing one tag costs discovery of plans that are probably under another.
			continue
		}
		for _, b := range found {
			if b.Slug == "" {
				continue
			}
			briefs[b.Slug] = b
			if !seen[b.Slug] {
				seen[b.Slug] = true
				order = append(order, b.Slug)
			}
		}
	}

	// Confirm each plan with a real fetch. This is what fills in the name, the pitch and
	// the true size for seed slugs, and it is affordable only because a plan is one
	// request: seventeen plans is seventeen calls, versus ~5,000 for the company packs.
	var list []leetcode.PlanBrief
	for _, slug := range order {
		if err := ctx.Err(); err != nil {
			emit(Progress{Done: len(list), Total: len(order), Note: "cancelled", Finished: true})
			return nil
		}

		// Retrying is an inner loop rather than a rewind of the outer index: `i--` inside
		// a range does nothing, since the range variable is reassigned every iteration.
		cancelled := false
	retry:
		for {
			plan, err := s.client.StudyPlan(ctx, slug)
			switch {
			case err == nil, errors.Is(err, leetcode.ErrPremiumRequired):
				// A gated plan still reports its name and size, so it belongs in the
				// picker with a PREMIUM note rather than being hidden. Hiding it would be
				// the app deciding the user cannot have something they may be paying for.
				list = append(list, plan.Brief())
			case errors.Is(err, leetcode.ErrRateLimited):
				emit(Progress{Done: len(list), Total: len(order), Note: "rate limited, waiting 30s"})
				select {
				case <-time.After(30 * time.Second):
					continue retry
				case <-ctx.Done():
					cancelled = true
				}
			case errors.Is(err, leetcode.ErrNotFound):
				// A retired or renamed slug. Skip it; TestLiveStudyPlanSlugs is what turns
				// this into a fixable fact rather than a silently shorter list.
				if b, ok := briefs[slug]; ok {
					list = append(list, b)
				}
			default:
				emit(Progress{Done: len(list), Total: len(order), Err: err, Finished: true})
				return fmt.Errorf("sync study plan registry at %s: %w", slug, err)
			}
			break
		}
		if cancelled {
			emit(Progress{Done: len(list), Total: len(order), Note: "cancelled", Finished: true})
			return nil
		}
		emit(Progress{Done: len(list), Total: len(order), Note: slug})
	}

	if err := s.store.UpsertPlans(ctx, list); err != nil {
		emit(Progress{Err: err, Finished: true})
		return err
	}
	if err := s.store.SetState(ctx, store.KeyPlansSyncedAt, time.Now().Format(time.RFC3339)); err != nil {
		emit(Progress{Done: len(list), Total: len(list), Err: err, Finished: true})
		return err
	}

	emit(Progress{Done: len(list), Total: len(list), Note: "done", Finished: true})
	return nil
}

// StudyPlan pulls one plan's problems.
//
// One request, no paging loop — the contrast with Pack is the point of D-031. The whole
// plan arrives or none of it does, so there is no half-written plan to misrepresent a
// curriculum.
func (s *Syncer) StudyPlan(ctx context.Context, slug string, out chan<- Progress) error {
	defer close(out)

	emit := func(p Progress) {
		p.Phase = PhasePlans
		select {
		case out <- p:
		case <-ctx.Done():
		}
	}

	emit(Progress{Note: slug})

	for {
		plan, err := s.client.StudyPlan(ctx, slug)
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
			return fmt.Errorf("sync study plan %s: %w", slug, err)
		}

		n := len(plan.Questions())
		if err := s.store.SetPlan(ctx, slug, plan); err != nil {
			emit(Progress{Err: err, Finished: true})
			return err
		}
		// Refresh the registry row from the detail, which is more authoritative than a
		// tag sweep's brief.
		if err := s.store.UpsertPlans(ctx, []leetcode.PlanBrief{plan.Brief()}); err != nil {
			emit(Progress{Done: n, Total: n, Err: err, Finished: true})
			return err
		}
		if err := s.store.SetState(ctx, store.PlanKey(slug), time.Now().Format(time.RFC3339)); err != nil {
			emit(Progress{Done: n, Total: n, Err: err, Finished: true})
			return err
		}

		emit(Progress{Done: n, Total: n, Note: "done", Finished: true})
		return nil
	}
}
