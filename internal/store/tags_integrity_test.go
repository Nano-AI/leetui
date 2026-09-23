package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// Exercise each partial payload writer against the public filter/search surface.
func tagWriters(s *Store) map[string]func([]leetcode.Tag) error {
	ctx := context.Background()
	return map[string]func([]leetcode.Tag) error{
		"detail": func(tags []leetcode.Tag) error {
			return s.SetDetail(ctx, &leetcode.Problem{Slug: "example", Title: "Example", Tags: tags})
		},
		"plan": func(tags []leetcode.Tag) error {
			return s.SetPlan(ctx, "course", leetcode.Plan{Groups: []leetcode.PlanGroup{{Questions: []leetcode.PlanQuestion{
				{Slug: "example", Title: "Example", Tags: tags},
			}}}})
		},
		"pack": func(tags []leetcode.Tag) error {
			return s.SetPack(ctx, "company", "all", []leetcode.PackQuestion{{Slug: "example", Title: "Example", Tags: tags}})
		},
	}
}

func assertTagQueries(t *testing.T, s *Store, tag string, want int) {
	t.Helper()
	for _, filter := range []Filter{{Tags: []string{tag}}, {Text: tag}} {
		rows, err := s.Query(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != want {
			t.Errorf("filter %+v returned %d rows, want %d", filter, len(rows), want)
		}
	}
}

func TestPartialPayloadTagsPersistAndRefresh(t *testing.T) {
	for _, kind := range []string{"detail", "plan", "pack"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			write := tagWriters(s)[kind]
			if err := write([]leetcode.Tag{{Slug: "arrays", Name: "Arrays"}}); err != nil {
				t.Fatal(err)
			}
			assertTagQueries(t, s, "arrays", 1)
			// A contest payload carries no tags; its reindex must use persisted ones.
			if err := s.SetContest(context.Background(), &leetcode.Contest{
				ContestBrief: leetcode.ContestBrief{Slug: "weekly"},
				Questions:    []leetcode.ContestQuestion{{Slug: "example", Title: "Example"}},
			}); err != nil {
				t.Fatal(err)
			}
			assertTagQueries(t, s, "arrays", 1)
			if err := write([]leetcode.Tag{{Slug: "graphs", Name: "Graphs"}}); err != nil {
				t.Fatal(err)
			}
			assertTagQueries(t, s, "arrays", 0)
			assertTagQueries(t, s, "graphs", 1)
			if err := write(nil); err != nil {
				t.Fatal(err)
			}
			assertTagQueries(t, s, "graphs", 1)
			if err := write([]leetcode.Tag{}); err != nil {
				t.Fatal(err)
			}
			assertTagQueries(t, s, "graphs", 0)
		})
	}
}

func TestPartialPayloadTagFailureRollsBack(t *testing.T) {
	for _, kind := range []string{"detail", "plan", "pack"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			if err := s.UpsertSummaries(ctx, []leetcode.ProblemSummary{{Slug: "example", Title: "Example", Tags: []leetcode.Tag{{Slug: "arrays", Name: "Arrays"}}}}); err != nil {
				t.Fatal(err)
			}
			_, err := s.DB().Exec(`CREATE TRIGGER reject_tag BEFORE INSERT ON problem_tags
				WHEN NEW.tag_slug = 'graphs' BEGIN SELECT RAISE(ABORT, 'tag write failed'); END`)
			if err != nil {
				t.Fatal(err)
			}
			if err := tagWriters(s)[kind]([]leetcode.Tag{{Slug: "graphs", Name: "Graphs"}}); err == nil {
				t.Error("tag storage failure reported success")
			}
			assertTagQueries(t, s, "arrays", 1)
			assertTagQueries(t, s, "graphs", 0)
			if kind == "plan" {
				if n, err := s.PlanCount(ctx, "course"); err != nil || n != 0 {
					t.Errorf("failed plan left %d links: %v", n, err)
				}
			}
			if kind == "pack" {
				if n, err := s.PackCount(ctx, "company", "all"); err != nil || n != 0 {
					t.Errorf("failed pack left %d links: %v", n, err)
				}
			}
		})
	}
}
