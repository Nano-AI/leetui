package syncer

import (
	"context"
	"errors"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// Editorial fetches and caches one problem's official solution.
//
// Like Detail, this is lazy: editorials are only pulled for problems the user opens one
// on. They are large — Two Sum's is several kilobytes of markdown — and most are never
// read, so syncing them in bulk would trade a lot of traffic for very little.
//
// A gated editorial is CACHED AS GATED and returned with ErrPremiumRequired. Storing the
// lock is what stops the pane re-requesting it on every keypress; the caller renders a
// lock state rather than an error (D-006).
func (s *Syncer) Editorial(ctx context.Context, slug string, force bool) (*store.Editorial, error) {
	if !force {
		if e, err := s.store.GetEditorial(ctx, slug); err == nil {
			if e.CanSee {
				return e, nil
			}
			// A gate cached before the user signed in with Premium is worth retrying
			// once a session exists — otherwise the lock would outlive the reason for it.
			if !s.client.Authenticated() {
				return e, leetcode.ErrPremiumRequired
			}
		}
	}

	sol, err := s.client.Editorial(ctx, slug)
	if sol != nil {
		// Cache both outcomes. A locked editorial is still a fact about the problem.
		_ = s.store.SetEditorial(ctx, slug, sol)
	}
	if err != nil {
		if errors.Is(err, leetcode.ErrPremiumRequired) && sol != nil {
			e, gerr := s.store.GetEditorial(ctx, slug)
			if gerr == nil {
				return e, err
			}
		}
		return nil, err
	}
	return s.store.GetEditorial(ctx, slug)
}
