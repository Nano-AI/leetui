package syncer

import (
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// Syncer pulls from LeetCode into the store.
type Syncer struct {
	client   *leetcode.Client
	store    *store.Store
	pageSize int
}

// New builds a Syncer. A pageSize of 0 uses 100, which is what the website requests.
func New(c *leetcode.Client, s *store.Store, pageSize int) *Syncer {
	if pageSize <= 0 {
		pageSize = 100
	}
	return &Syncer{client: c, store: s, pageSize: pageSize}
}
