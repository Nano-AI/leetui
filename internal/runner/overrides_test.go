package runner

import "testing"

// The NoLocal list is a claim about specific problems, so it is worth asserting rather
// than trusting: gating on metaData.manual instead would have caught number-of-islands,
// whose description is exact and which runs today.
func TestNoLocalIsNarrow(t *testing.T) {
	cannot := []string{
		"clone-graph",
		"construct-quad-tree",
		"encode-n-ary-tree-to-binary-tree",
		"linked-list-cycle",
		"inorder-successor-in-bst",
		"read-n-characters-given-read4",
		"read-n-characters-given-read4-ii-call-multiple-times",
	}
	for _, slug := range cannot {
		if RuleFor(slug).NoLocal == "" {
			t.Errorf("%s should be marked NoLocal", slug)
		}
	}

	// Marked manual by LeetCode, described accurately, and generating a driver for it
	// works. Gating on the flag would break it.
	for _, slug := range []string{"number-of-islands", "two-sum", "merge-k-sorted-lists"} {
		if r := RuleFor(slug).NoLocal; r != "" {
			t.Errorf("%s should run locally, got NoLocal=%q", slug, r)
		}
	}
}
