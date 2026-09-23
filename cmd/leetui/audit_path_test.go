package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func TestPathNeedsOnlyCachedSummary(t *testing.T) {
	a := offlineContestApp(t)
	if err := a.store.UpsertSummaries(context.Background(), []leetcode.ProblemSummary{{Slug: "two-sum", FrontendID: "1", Title: "Two Sum"}}); err != nil {
		t.Fatal(err)
	}
	if code, err := runPath(a, []string{"two-sum"}); code != exitOK || err != nil {
		t.Fatalf("path fetched a statement instead of using cached ID: %d, %v", code, err)
	}
}

func TestContestRejectsExplicitBlankProblem(t *testing.T) {
	a := offlineContestApp(t)
	code, err := runContest(a, []string{"submit", "weekly-test", "   "})
	if code != exitProblem || err == nil || !strings.Contains(err.Error(), "problem") {
		t.Fatalf("explicit blank problem reached account/submission workflow: %d, %v", code, err)
	}
}

func TestDispatchEmptyCommandReturnsUsageError(t *testing.T) {
	if code := dispatch([]string{""}); code != exitProblem {
		t.Fatalf("empty command returned %d, want 2", code)
	}
}
