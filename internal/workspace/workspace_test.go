package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func testWorkspace(t *testing.T) Workspace {
	t.Helper()
	w, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}
	return w
}

func sampleProblem() Problem {
	return Problem{
		ID: 146, Slug: "lru-cache", Title: "LRU Cache",
		Statement:  "Design a data structure that follows the constraints of an LRU cache.\n",
		Difficulty: "Medium",
		URL:        "https://leetcode.com/problems/lru-cache/",
	}
}

func TestFolderName(t *testing.T) {
	cases := []struct {
		id         int
		slug, want string
	}{
		{1, "two-sum", "0001-two-sum"},
		{146, "lru-cache", "0146-lru-cache"},
		{1650, "lowest-common-ancestor-iii", "1650-lowest-common-ancestor-iii"},
		{0, "lcp-01-guess-numbers", "0000-lcp-01-guess-numbers"},
		// The slug is remote data, so it is sanitised rather than trusted.
		{7, "Reverse Integer!", "0007-reverse-integer"},
		{8, "../../etc/passwd", "0008-etc-passwd"},
		{9, "", "0009-problem"},
	}
	for _, tc := range cases {
		if got := FolderName(tc.id, tc.slug); got != tc.want {
			t.Errorf("FolderName(%d, %q) = %q, want %q", tc.id, tc.slug, got, tc.want)
		}
	}
}

// TestFolderNameCannotEscape is the security check behind the sanitiser: a hostile slug
// must never produce a path outside the workspace root.
func TestFolderNameCannotEscape(t *testing.T) {
	w := testWorkspace(t)
	for _, slug := range []string{"../evil", "../../etc/passwd", "a/b/c", "..", "./."} {
		dir := w.Dir(1, slug)
		if !strings.HasPrefix(filepath.Clean(dir), filepath.Clean(w.Root)+string(filepath.Separator)) {
			t.Errorf("slug %q escaped the workspace: %s", slug, dir)
		}
	}
}
