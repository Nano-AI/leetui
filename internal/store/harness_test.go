package store

import (
	"path/filepath"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	// A file rather than :memory: so migrations and WAL run the same path as production.
	s, err := OpenPath(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sample() []leetcode.ProblemSummary {
	return []leetcode.ProblemSummary{
		{
			FrontendID: "1", Title: "Two Sum", Slug: "two-sum",
			Difficulty: leetcode.Easy, AcRate: 57.9, Status: leetcode.StatusAccepted,
			Tags: []leetcode.Tag{{Name: "Array", Slug: "array"}, {Name: "Hash Table", Slug: "hash-table"}},
		},
		{
			FrontendID: "146", Title: "LRU Cache", Slug: "lru-cache",
			Difficulty: leetcode.Medium, AcRate: 42.7,
			Tags: []leetcode.Tag{{Name: "Design", Slug: "design"}, {Name: "Hash Table", Slug: "hash-table"}},
		},
		{
			FrontendID: "42", Title: "Trapping Rain Water", Slug: "trapping-rain-water",
			Difficulty: leetcode.Hard, AcRate: 61.2, Status: leetcode.StatusAttempted,
			PaidOnly: true,
			Tags:     []leetcode.Tag{{Name: "Array", Slug: "array"}, {Name: "Two Pointers", Slug: "two-pointers"}},
		},
	}
}
