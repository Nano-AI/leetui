package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func TestSearchSurvivesCollectionRefresh(t *testing.T) {
	ctx := context.Background()
	for _, refresh := range []string{"summaries", "contest", "detail"} {
		t.Run(refresh, func(t *testing.T) {
			s := testStore(t)
			if err := s.UpsertSummaries(ctx, sample()); err != nil {
				t.Fatal(err)
			}
			p := &leetcode.Problem{Slug: "two-sum", Title: "Two Sum", Content: "<p>Distinctive prose: zephyr</p>"}
			if err := s.SetDetail(ctx, p); err != nil {
				t.Fatal(err)
			}
			var err error
			switch refresh {
			case "summaries":
				err = s.UpsertSummaries(ctx, sample())
			case "contest":
				err = s.SetContest(ctx, &leetcode.Contest{ContestBrief: leetcode.ContestBrief{Slug: "weekly"}, Questions: []leetcode.ContestQuestion{{Slug: p.Slug, Title: p.Title}}})
			case "detail":
				err = s.SetDetail(ctx, p)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{"zephyr", "array"} {
				rows, err := s.Query(ctx, Filter{Text: text})
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, r := range rows {
					found = found || r.Slug == p.Slug
				}
				if !found {
					t.Errorf("%q no longer finds %s after %s", text, p.Slug, refresh)
				}
			}
		})
	}
}

func TestCollectionSeedsAreSearchable(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"plan", "pack"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			var err error
			if kind == "plan" {
				err = s.SetPlan(ctx, "course", leetcode.Plan{Groups: []leetcode.PlanGroup{{Questions: []leetcode.PlanQuestion{{Slug: "fresh", Title: "Zephyr"}}}}})
			} else {
				err = s.SetPack(ctx, "company", "all", []leetcode.PackQuestion{{Slug: "fresh", Title: "Zephyr"}})
			}
			if err != nil {
				t.Fatal(err)
			}
			rows, err := s.Query(ctx, Filter{Text: "zephyr"})
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 {
				t.Fatalf("search found %d seeded problems, want 1", len(rows))
			}
		})
	}
}
