package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func TestContestScheduleCanMoveEarlierAndShorten(t *testing.T) {
	ctx := context.Background()
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "schedule", true: "detail"}[detail], func(t *testing.T) {
			s := testStore(t)
			if err := s.UpsertContests(ctx, []leetcode.ContestBrief{{Slug: "weekly", StartTime: 2000, Duration: 5400}}); err != nil {
				t.Fatal(err)
			}
			for _, brief := range []leetcode.ContestBrief{
				{Slug: "weekly", StartTime: 1000, Duration: 3600},
				{Slug: "weekly"}, // incomplete metadata must still preserve known values
			} {
				var err error
				if detail {
					err = s.SetContest(ctx, &leetcode.Contest{ContestBrief: brief})
				} else {
					err = s.UpsertContests(ctx, []leetcode.ContestBrief{brief})
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			c, err := s.ContestOf(ctx, "weekly")
			if err != nil {
				t.Fatal(err)
			}
			if c.StartTime != 1000 || c.Duration != 3600 {
				t.Fatalf("schedule retained obsolete clock: %+v", c)
			}
		})
	}
}

func TestPlanSizeCanShrink(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	for _, size := range []int{150, 100, 0} {
		if err := s.UpsertPlans(ctx, []leetcode.PlanBrief{{Slug: "course", QuestionNum: size}}); err != nil {
			t.Fatal(err)
		}
	}
	plans, err := s.Plans(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].QuestionNum != 100 {
		t.Fatalf("obsolete progress denominator: %+v", plans)
	}
}
