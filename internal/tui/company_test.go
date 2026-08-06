package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// seedRegistry puts a company registry in the model's store, standing in for the one
// request CompanyRegistry would make.
func seedRegistry(t *testing.T, m Model) {
	t.Helper()
	err := m.store.UpsertCompanies(context.Background(), []leetcode.Company{
		{Slug: "google", Name: "Google", QuestionCount: 2335},
		{Slug: "facebook", Name: "Meta", QuestionCount: 1402},
		{Slug: "bloomberg", Name: "Bloomberg", QuestionCount: 1209},
	})
	if err != nil {
		t.Fatalf("seed registry: %v", err)
	}
}

func TestCompanyBrowserFiltersAsYouType(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedRegistry(t, m)

	m = drive(t, m, key("c"))
	if m.mode != modeCompany {
		t.Fatalf("c did not open the company browser; mode is %v", m.mode)
	}
	if len(m.companies) != 3 {
		t.Fatalf("loaded %d companies, want 3", len(m.companies))
	}

	// Typing narrows directly — no search key first. There is only one thing to do with
	// a keystroke in a list of 984 companies.
	m = drive(t, m, key("m"), key("e"), key("t"))
	rows := m.visibleCompanies()
	if len(rows) != 1 || rows[0].Slug != "facebook" {
		t.Fatalf(`typing "met" left %+v, want just Meta`, rows)
	}

	out := m.View()
	if !strings.Contains(out, "Meta") {
		t.Error("the filtered company is not on screen")
	}
	if strings.Contains(out, "Bloomberg") {
		t.Error("a filtered-out company is still on screen")
	}
}

// TestCompanyBrowserReanchorsCursor guards a real crash shape: narrow the list while the
// cursor sits on row 40 and enter would read past the end.
func TestCompanyBrowserReanchorsCursor(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedRegistry(t, m)

	// Two drive calls, not one: the registry arrives on a command, and a human presses
	// c, sees the list, and only then moves. Driving both in one round would test a
	// cursor moving through a list that has not loaded.
	m = drive(t, m, key("c"))
	m = drive(t, m, key("down"), key("down"))
	if m.companyIdx != 2 {
		t.Fatalf("cursor is at %d, want 2", m.companyIdx)
	}

	m = drive(t, m, key("m"))
	if m.companyIdx != 0 {
		t.Errorf("cursor stayed at %d after the list shrank; want 0", m.companyIdx)
	}
}

func TestCompanyPickLeadsToTimeframe(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedRegistry(t, m)

	m = drive(t, m, key("c"))
	m = drive(t, m, key("enter"))

	if m.picking != pickTimeframe {
		t.Fatalf("choosing a company did not open the timeframe picker; picking is %v", m.picking)
	}
	if m.packChoice.Slug != "google" {
		t.Fatalf("picked %q, want google — the registry is ordered by size", m.packChoice.Slug)
	}

	// All five of LeetCode's windows, and no others.
	rows := m.pickerRows()
	if len(rows) != 5 {
		t.Fatalf("timeframe picker has %d rows, want 5", len(rows))
	}
	out := m.View()
	// Frame titles are set in the bezel uppercased; row labels are prose.
	for _, want := range []string{"ASKED BY GOOGLE", "last 30 days", "all time"} {
		if !strings.Contains(out, want) {
			t.Errorf("timeframe picker is missing %q", want)
		}
	}
}

func TestPackFiltersTheBoard(t *testing.T) {
	m := boot(t, true, 120, 32)
	seedRegistry(t, m)
	ctx := context.Background()

	err := m.store.SetPack(ctx, "google", "all", []leetcode.PackQuestion{
		{Slug: "lru-cache", Title: "LRU Cache", FrontendID: "146",
			Difficulty: leetcode.Medium, Frequency: 0.3},
		{Slug: "two-sum", Title: "Two Sum", FrontendID: "1",
			Difficulty: leetcode.Easy, Frequency: 0.9},
	})
	if err != nil {
		t.Fatalf("seed pack: %v", err)
	}

	// c, then enter (Google), then down four times to reach "all time", then enter.
	m = drive(t, m, key("c"))
	keys := []tea.Msg{key("enter")}
	for i := 0; i < 4; i++ {
		keys = append(keys, key("down"))
	}
	m = drive(t, m, append(keys, key("enter"))...)

	if !m.pack.Active() || m.pack.Company != "google" {
		t.Fatalf("pack is %+v, want google", m.pack)
	}
	if m.pack.Timeframe != leetcode.AllTime {
		t.Fatalf("timeframe is %q, want all", m.pack.Timeframe)
	}
	if len(m.rows) != 2 {
		t.Fatalf("board shows %d rows, want the 2 in the pack", len(m.rows))
	}
	// Frequency order is what makes a pack more useful than a tag filter.
	if m.rows[0].Slug != "two-sum" {
		t.Errorf("top row is %q, want two-sum — the pack sorts by frequency", m.rows[0].Slug)
	}

	out := m.View()
	for _, want := range []string{"Google", "all time"} {
		if !strings.Contains(out, want) {
			t.Errorf("the active pack is not announced on screen: missing %q", want)
		}
	}

	// esc clears it, and the whole problem set comes back.
	m = drive(t, m, key("esc"))
	if m.pack.Active() {
		t.Error("esc left the pack in place")
	}
	if len(m.rows) != 4 {
		t.Errorf("clearing the pack left %d rows, want all 4 seeded problems", len(m.rows))
	}
}

func TestPremiumFilterCycles(t *testing.T) {
	m := boot(t, true, 120, 32)

	m = drive(t, m, key("p"))
	if m.filter.PaidOnly == nil || !*m.filter.PaidOnly {
		t.Fatal("first p did not filter to premium-only")
	}
	if len(m.rows) != 1 {
		t.Errorf("premium-only shows %d rows, want the 1 seeded paid problem", len(m.rows))
	}

	m = drive(t, m, key("p"))
	if m.filter.PaidOnly == nil || *m.filter.PaidOnly {
		t.Fatal("second p did not filter to free-only")
	}
	if len(m.rows) != 3 {
		t.Errorf("free-only shows %d rows, want 3", len(m.rows))
	}

	m = drive(t, m, key("p"))
	if m.filter.PaidOnly != nil {
		t.Error("third p did not clear the filter")
	}
}
