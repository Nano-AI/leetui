package syncer

import (
	"context"
	"errors"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// Detail fetches and stores one problem's full record.
//
// This is the lazy half of D-009: the list sync is fast and shallow, statements arrive
// as the user opens them. Already-fetched problems are returned from the store without
// a network call unless force is set.
func (s *Syncer) Detail(ctx context.Context, slug string, force bool) (*store.Detail, error) {
	if !force {
		if d, err := s.store.Get(ctx, slug); err == nil && d.HasDetail {
			return d, nil
		}
	}

	p, err := s.client.Question(ctx, slug)
	if err != nil {
		// A premium-gated problem still has a usable row; the caller renders a lock
		// state rather than an error (D-006).
		if errors.Is(err, leetcode.ErrPremiumRequired) {
			if d, gerr := s.store.Get(ctx, slug); gerr == nil {
				return d, err
			}
		}
		return nil, err
	}

	if err := s.store.SetDetail(ctx, p); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, slug)
}

// Account refreshes the signed-in user's status, including Premium.
//
// A signed-out session is not an error: it stores empty values so the rail shows a free
// account rather than a stale premium badge.
func (s *Syncer) Account(ctx context.Context) (leetcode.UserStatus, error) {
	st, err := s.client.Status(ctx)
	if err != nil {
		return st, err
	}
	_ = s.store.SetState(ctx, store.KeyUsername, st.Username)
	premium := "0"
	if st.IsPremium {
		premium = "1"
	}
	_ = s.store.SetState(ctx, store.KeyIsPremium, premium)
	return st, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
