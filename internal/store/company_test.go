package store

import (
	"context"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func seedCompanies(t *testing.T, s *Store) {
	t.Helper()
	err := s.UpsertCompanies(context.Background(), []leetcode.Company{
		{Slug: "google", Name: "Google", QuestionCount: 2335},
		{Slug: "facebook", Name: "Meta", QuestionCount: 1402},
		{Slug: "amazon", Name: "Amazon", QuestionCount: 1998},
	})
	if err != nil {
		t.Fatalf("upsert companies: %v", err)
	}
}

func TestCompaniesOrderBySize(t *testing.T) {
	s := testStore(t)
	seedCompanies(t, s)

	got, err := s.Companies(context.Background(), "")
	if err != nil {
		t.Fatalf("Companies: %v", err)
	}
	// Largest first: in a 984-row list the companies people prepare for are the big ones,
	// and alphabetical order would bury them.
	want := []string{"google", "amazon", "facebook"}
	if len(got) != len(want) {
		t.Fatalf("got %d companies, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Slug != w {
			t.Errorf("position %d = %q, want %q", i, got[i].Slug, w)
		}
	}
}

// TestCompanyFilterMatchesDisplayName is the case that would trip a slug-only filter:
// Meta's slug is still "facebook", so typing what is on screen has to work.
func TestCompanyFilterMatchesDisplayName(t *testing.T) {
	s := testStore(t)
	seedCompanies(t, s)

	got, err := s.Companies(context.Background(), "meta")
	if err != nil {
		t.Fatalf("Companies: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "facebook" {
		t.Fatalf(`Companies("meta") = %+v, want just facebook`, got)
	}

	got, err = s.Companies(context.Background(), "facebook")
	if err != nil {
		t.Fatalf("Companies: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Meta" {
		t.Fatalf(`Companies("facebook") = %+v, want just Meta`, got)
	}
}

func TestSetPackFiltersAndRanks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedCompanies(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	pack := []leetcode.PackQuestion{
		{Slug: "lru-cache", Title: "LRU Cache", FrontendID: "146",
			Difficulty: leetcode.Medium, Frequency: 0.2},
		{Slug: "two-sum", Title: "Two Sum", FrontendID: "1",
			Difficulty: leetcode.Easy, Frequency: 0.9},
	}
	if err := s.SetPack(ctx, "google", "three-months", pack); err != nil {
		t.Fatalf("SetPack: %v", err)
	}

	rows, err := s.Query(ctx, Filter{
		Companies: []string{"google"},
		Timeframe: "three-months",
		Sort:      "frequency",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	// Frequency order is the point of a pack: the problem this company asks most comes
	// first, even though its ID is lower.
	if rows[0].Slug != "two-sum" {
		t.Errorf("top row is %q, want two-sum (frequency 0.9 beats 0.2)", rows[0].Slug)
	}

	// A different timeframe of the same company is a different pack and must be empty.
	rows, err = s.Query(ctx, Filter{Companies: []string{"google"}, Timeframe: "thirty-days"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("thirty-days returned %d rows, want 0 — timeframes must not bleed", len(rows))
	}
}

// TestSetPackSeedsUnknownProblems covers the premium case: a company list is sometimes
// the first place a problem appears locally, and dropping those entries would silently
// shorten the pack.
func TestSetPackSeedsUnknownProblems(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedCompanies(t, s)

	err := s.SetPack(ctx, "google", "all", []leetcode.PackQuestion{
		{Slug: "meeting-rooms-ii", Title: "Meeting Rooms II", FrontendID: "253",
			Difficulty: leetcode.Medium, PaidOnly: true, Frequency: 0.7},
	})
	if err != nil {
		t.Fatalf("SetPack: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Companies: []string{"google"}})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0].Title != "Meeting Rooms II" {
		t.Fatalf("pack did not seed the unknown problem: %+v", rows)
	}
	if !rows[0].PaidOnly {
		t.Error("seeded problem lost its premium flag")
	}
}

// TestSetPackReplacesItsOwnTimeframe guards the refresh path: a problem a company stopped
// asking must leave the list, and the other timeframes must survive.
func TestSetPackReplacesItsOwnTimeframe(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedCompanies(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	first := []leetcode.PackQuestion{
		{Slug: "two-sum", FrontendID: "1", Title: "Two Sum", Frequency: 0.5},
		{Slug: "lru-cache", FrontendID: "146", Title: "LRU Cache", Frequency: 0.4},
	}
	if err := s.SetPack(ctx, "google", "thirty-days", first); err != nil {
		t.Fatalf("SetPack: %v", err)
	}
	if err := s.SetPack(ctx, "google", "all", first); err != nil {
		t.Fatalf("SetPack all: %v", err)
	}

	// Google stopped asking LRU Cache in the last thirty days.
	if err := s.SetPack(ctx, "google", "thirty-days", first[:1]); err != nil {
		t.Fatalf("SetPack refresh: %v", err)
	}

	n, err := s.PackCount(ctx, "google", "thirty-days")
	if err != nil {
		t.Fatalf("PackCount: %v", err)
	}
	if n != 1 {
		t.Errorf("thirty-days has %d problems after refresh, want 1", n)
	}
	if n, err = s.PackCount(ctx, "google", "all"); err != nil || n != 2 {
		t.Errorf("all-time pack is %d (err %v) — refreshing one timeframe wiped another", n, err)
	}
}

func TestCompanySyncedCount(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedCompanies(t, s)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	// The same problem in two timeframes is one problem, not two.
	q := []leetcode.PackQuestion{{Slug: "two-sum", FrontendID: "1", Title: "Two Sum"}}
	for _, tf := range []string{"thirty-days", "all"} {
		if err := s.SetPack(ctx, "google", tf, q); err != nil {
			t.Fatalf("SetPack %s: %v", tf, err)
		}
	}

	got, err := s.Companies(ctx, "google")
	if err != nil {
		t.Fatalf("Companies: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d companies, want 1", len(got))
	}
	if got[0].Synced != 1 {
		t.Errorf("Synced = %d, want 1 distinct problem across two timeframes", got[0].Synced)
	}
}
