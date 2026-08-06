package store

import (
	"context"
	"testing"
)

func TestSearch(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		query string
		want  string // expected slug of the first hit; "" means no results
	}{
		{"two sum", "two-sum"},
		{"lru", "lru-cache"},
		{"cach", "lru-cache"}, // prefix matching: search narrows as you type
		{"rain", "trapping-rain-water"},
		{"design", "lru-cache"}, // tags are indexed too
		{"zzzznothing", ""},
	}

	for _, tc := range cases {
		rows, err := s.Query(ctx, Filter{Text: tc.query})
		if err != nil {
			t.Errorf("search %q: %v", tc.query, err)
			continue
		}
		if tc.want == "" {
			if len(rows) != 0 {
				t.Errorf("search %q returned %d rows, want none", tc.query, len(rows))
			}
			continue
		}
		if len(rows) == 0 {
			t.Errorf("search %q returned nothing, want %s", tc.query, tc.want)
			continue
		}
		if rows[0].Slug != tc.want {
			t.Errorf("search %q top hit = %s, want %s", tc.query, rows[0].Slug, tc.want)
		}
	}
}

// TestSearchHandlesHostileInput asserts user text never reaches FTS5 as syntax.
// A stray quote or operator must degrade to a normal search, not a SQL error.
func TestSearchHandlesHostileInput(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{
		`"`, `two"sum`, `NEAR(a b)`, `a OR b`, `*`, `-sum`, `col:val`, `((`, `''`, `^`,
	} {
		if _, err := s.Query(ctx, Filter{Text: q}); err != nil {
			t.Errorf("search %q errored: %v", q, err)
		}
	}
}

func TestFilters(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	paid := true
	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"difficulty", Filter{Difficulty: []string{"Easy", "Hard"}}, 2},
		{"solved", Filter{Status: "ac"}, 1},
		{"attempted", Filter{Status: "notac"}, 1},
		{"todo", Filter{Status: "todo"}, 1},
		{"single tag", Filter{Tags: []string{"hash-table"}}, 2},
		{"all tags must match", Filter{Tags: []string{"hash-table", "design"}}, 1},
		{"impossible tag combo", Filter{Tags: []string{"design", "two-pointers"}}, 0},
		{"premium only", Filter{PaidOnly: &paid}, 1},
		{"search plus filter", Filter{Text: "a", Difficulty: []string{"Hard"}}, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := s.Query(ctx, tc.f)
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if len(rows) != tc.want {
				slugs := make([]string, len(rows))
				for i, r := range rows {
					slugs[i] = r.Slug
				}
				t.Errorf("got %d rows %v, want %d", len(rows), slugs, tc.want)
			}
		})
	}
}

func TestSortDifficultyIsSemantic(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	rows, err := s.Query(ctx, Filter{Sort: "difficulty"})
	if err != nil {
		t.Fatal(err)
	}
	// Alphabetical would give Easy, Hard, Medium. Semantic order is Easy, Medium, Hard.
	want := []string{"Easy", "Medium", "Hard"}
	for i, w := range want {
		if rows[i].Difficulty != w {
			t.Errorf("position %d = %s, want %s", i, rows[i].Difficulty, w)
		}
	}
}
