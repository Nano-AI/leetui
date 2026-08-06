package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/grootbeat/leetui/internal/leetcode"
)

func TestDetailRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	p := &leetcode.Problem{
		QuestionID: "1", FrontendID: "1", Title: "Two Sum", Slug: "two-sum",
		Content:        "<p>Given an array of <strong>integers</strong> nums, return indices.</p>",
		MetaData:       `{"name":"twoSum","params":[{"name":"nums","type":"integer[]"}],"return":{"type":"integer[]"}}`,
		SampleTestCase: "[2,7,11,15]\n9",
		Hints:          []string{"Use a hash map.", "One pass is enough."},
		Snippets: []leetcode.CodeSnippet{
			{Lang: "Python3", LangSlug: "python3", Code: "class Solution:\n    pass"},
			{Lang: "Go", LangSlug: "golang", Code: "func twoSum() {}"},
		},
		Tags: []leetcode.Tag{{Name: "Array", Slug: "array"}},
	}
	if err := s.SetDetail(ctx, p); err != nil {
		t.Fatalf("set detail: %v", err)
	}

	d, err := s.Get(ctx, "two-sum")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !d.HasDetail {
		t.Error("HasDetail is false after SetDetail")
	}
	if d.QuestionID != "1" {
		t.Errorf("QuestionID = %q, want \"1\"", d.QuestionID)
	}
	if len(d.Snippets) != 2 || d.Snippets["golang"] == "" {
		t.Errorf("snippets = %v", d.Snippets)
	}
	if len(d.Hints) != 2 {
		t.Errorf("hints = %v, want 2", d.Hints)
	}

	// The statement is now searchable, and HTML tags must not be.
	rows, err := s.Query(ctx, Filter{Text: "indices"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("statement search: %d rows, err %v", len(rows), err)
	}
	if rows, _ := s.Query(ctx, Filter{Text: "strong"}); len(rows) != 0 {
		t.Error("HTML tag names leaked into the search index")
	}

	// A later list sync must not wipe the fetched statement.
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}
	d2, err := s.Get(ctx, "two-sum")
	if err != nil {
		t.Fatal(err)
	}
	if !d2.HasDetail || d2.Content == "" {
		t.Error("list sync wiped previously fetched detail")
	}
}

func TestStats(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	st, err := s.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 3 || st.Solved != 1 || st.Attempted != 1 || st.Easy != 1 {
		t.Errorf("stats = %+v", st)
	}
}

func TestSyncState(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	if v, err := s.GetState(ctx, KeyProblemsCursor); err != nil || v != "" {
		t.Fatalf("missing key = %q, %v; want empty and no error", v, err)
	}
	if err := s.SetState(ctx, KeyProblemsCursor, "300"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetState(ctx, KeyProblemsCursor, "400"); err != nil {
		t.Fatal(err)
	}
	v, err := s.GetState(ctx, KeyProblemsCursor)
	if err != nil || Atoi(v) != 400 {
		t.Errorf("cursor = %q, %v; want 400", v, err)
	}
}

// TestMigrateIsIdempotent asserts reopening an existing database does not re-run
// migrations or fail.
func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	ctx := context.Background()

	s1, err := OpenPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}
	s1.Close()

	s2, err := OpenPath(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	n, err := s2.Count(ctx)
	if err != nil || n != 3 {
		t.Errorf("count after reopen = %d, %v; want 3", n, err)
	}
}
