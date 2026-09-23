package store

import (
	"context"
	"testing"
)

func TestSetAndReadMarks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	if err := s.SetMark(ctx, "two-sum", MarkUp, "classic"); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "lru-cache", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	got, err := s.MarkOf(ctx, "two-sum")
	if err != nil || got != MarkUp {
		t.Errorf("MarkOf(two-sum) = %q (err %v), want up", got, err)
	}
	// An unmarked problem is no opinion, not an error.
	if got, err = s.MarkOf(ctx, "trapping-rain-water"); err != nil || got != MarkNone {
		t.Errorf("MarkOf on an unmarked problem = %q (err %v), want empty", got, err)
	}

	ups, err := s.Marks(ctx, MarkUp)
	if err != nil {
		t.Fatalf("Marks: %v", err)
	}
	if len(ups) != 1 || ups[0].Slug != "two-sum" || ups[0].Note != "classic" {
		t.Errorf("Marks(up) = %+v, want just two-sum with its note", ups)
	}

	all, err := s.Marks(ctx, MarkNone)
	if err != nil {
		t.Fatalf("Marks: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Marks(none) returned %d, want both directions", len(all))
	}
}

// TestSetMarkRejectsNonDirections: clearing goes through ClearMark, so an empty or bogus
// value must not be able to write a half-set row.
func TestSetMarkRejectsNonDirections(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	for _, bad := range []Mark{MarkNone, MarkAny, Mark("sideways")} {
		if err := s.SetMark(ctx, "two-sum", bad, ""); err == nil {
			t.Errorf("SetMark accepted %q; only up and down are storable", bad)
		}
	}
	if n, _ := s.Marks(ctx, MarkNone); len(n) != 0 {
		t.Errorf("a rejected mark still wrote %d rows", len(n))
	}
}

// TestMarkIsIdempotentAndFlips is the contract an agent depends on: no read before write,
// and re-marking is not an error.
func TestMarkIsIdempotentAndFlips(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := s.SetMark(ctx, "two-sum", MarkUp, ""); err != nil {
			t.Fatalf("SetMark pass %d: %v", i, err)
		}
	}
	all, _ := s.Marks(ctx, MarkNone)
	if len(all) != 1 {
		t.Fatalf("marking three times made %d rows, want 1", len(all))
	}

	// Flipping a verdict replaces it rather than adding a second.
	if err := s.SetMark(ctx, "two-sum", MarkDown, ""); err != nil {
		t.Fatalf("SetMark flip: %v", err)
	}
	got, _ := s.MarkOf(ctx, "two-sum")
	if got != MarkDown {
		t.Errorf("after flipping, mark is %q, want down", got)
	}

	// Clearing something unmarked is a success, not an error.
	if err := s.ClearMark(ctx, "never-marked"); err != nil {
		t.Errorf("clearing an unmarked problem errored: %v", err)
	}
}

// TestSetMarkKeepsAnExistingNote: a bulk agent pass with no note must not wipe a reason a
// human wrote.
func TestSetMarkKeepsAnExistingNote(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SetMark(ctx, "two-sum", MarkUp, "the hash-map insight"); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "two-sum", MarkUp, ""); err != nil {
		t.Fatalf("SetMark again: %v", err)
	}

	all, _ := s.Marks(ctx, MarkUp)
	if len(all) != 1 || all[0].Note != "the hash-map insight" {
		t.Errorf("an empty note erased the original: %+v", all)
	}

	// A real note does replace it.
	if err := s.SetMark(ctx, "two-sum", MarkUp, "revised"); err != nil {
		t.Fatalf("SetMark revise: %v", err)
	}
	all, _ = s.Marks(ctx, MarkUp)
	if all[0].Note != "revised" {
		t.Errorf("note is %q, want revised", all[0].Note)
	}
}

// TestMarkFilterIsOrthogonalToStatus is the query the whole feature exists for: solved,
// AND worth doing again.
func TestMarkFilterIsOrthogonalToStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// two-sum is accepted in sample(); lru-cache is untouched.
	if err := s.SetMark(ctx, "two-sum", MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "lru-cache", MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Mark: MarkUp, Status: "ac"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0].Slug != "two-sum" {
		t.Fatalf("solved-and-important gave %+v, want just two-sum", rows)
	}

	// MarkAny matches either direction and must never be written to the table.
	if err := s.SetMark(ctx, "trapping-rain-water", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	rows, err = s.Query(ctx, Filter{Mark: MarkAny})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("MarkAny matched %d rows, want all 3 marked", len(rows))
	}
}

// TestMarkSortPutsUnmarkedInTheMiddle: burying thousands of unmarked problems under a
// handful written off would make the sort useless on a barely-marked board.
func TestMarkSortPutsUnmarkedInTheMiddle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// lru-cache is #146 and trapping-rain-water is #42, so ID order alone would not
	// produce the expected result.
	if err := s.SetMark(ctx, "lru-cache", MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "trapping-rain-water", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Sort: "mark"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []string{"lru-cache", "two-sum", "trapping-rain-water"}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for i, w := range want {
		if rows[i].Slug != w {
			t.Errorf("position %d is %q, want %q (up, unmarked, down)", i, rows[i].Slug, w)
		}
	}
}

// TestUnimportantSinksButIsNeverRemoved is the rule that matters most about `-`
// (D-032a): it demotes, it does not delete. The row stays in every result set, at the
// bottom of it.
func TestUnimportantSinksButIsNeverRemoved(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// two-sum is #1 and would lead any ID-ordered board.
	if err := s.SetMark(ctx, "two-sum", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := s.Query(ctx, Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("the unimportant problem was removed: %d rows, want 3", len(rows))
	}
	if rows[len(rows)-1].Slug != "two-sum" {
		t.Errorf("last row is %q, want two-sum at the bottom", rows[len(rows)-1].Slug)
	}
	// The ones above it keep their own order.
	if rows[0].Slug != "trapping-rain-water" || rows[1].Slug != "lru-cache" {
		t.Errorf("the undemoted rows lost their ID order: %q, %q", rows[0].Slug, rows[1].Slug)
	}
}

// TestDemotionAppliesToEverySort: the rule holds whatever else is ordering the board, or
// a plan and a company pack would each need their own version of it.
func TestDemotionAppliesToEverySort(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	if err := s.SetMark(ctx, "two-sum", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	// two-sum sorts first by title ("Two Sum" is not first alphabetically, but it is the
	// lowest ID and the highest acceptance) — under each sort it must still land last.
	for _, sort := range []string{"", "title", "acrate", "difficulty", "todo"} {
		rows, err := s.Query(ctx, Filter{Sort: sort})
		if err != nil {
			t.Fatalf("Query(%q): %v", sort, err)
		}
		if len(rows) != 3 {
			t.Errorf("sort %q returned %d rows, want 3", sort, len(rows))
			continue
		}
		if rows[len(rows)-1].Slug != "two-sum" {
			t.Errorf("sort %q put %q last, want two-sum demoted", sort, rows[len(rows)-1].Slug)
		}
	}
}

// TestDemotionChangesNothingWithoutMarks: the prefix must be inert on a board nobody has
// marked, or every existing ordering quietly shifts.
func TestDemotionChangesNothingWithoutMarks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}

	rows, err := s.Query(ctx, Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []string{"two-sum", "trapping-rain-water", "lru-cache"}
	for i, w := range want {
		if rows[i].Slug != w {
			t.Fatalf("unmarked board is ordered %+v, want plain ID order", rows)
		}
	}
}

// TestUnimportantStaysSearchable: demotion must not take a problem out of search, or "-"
// becomes a delete by another route.
func TestUnimportantStaysSearchable(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	if err := s.SetMark(ctx, "two-sum", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := s.Query(ctx, Filter{Text: "two sum"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("a problem marked unimportant vanished from search")
	}
}

// TestImportantFloatsToTheTop is the other half of the rule (D-032b): `+` promotes in
// every sort, the way `-` demotes.
func TestImportantFloatsToTheTop(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// lru-cache is #146, last in ID order.
	if err := s.SetMark(ctx, "lru-cache", MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	for _, sort := range []string{"", "title", "acrate", "difficulty"} {
		rows, err := s.Query(ctx, Filter{Sort: sort})
		if err != nil {
			t.Fatalf("Query(%q): %v", sort, err)
		}
		if rows[0].Slug != "lru-cache" {
			t.Errorf("sort %q leads with %q, want lru-cache floated", sort, rows[0].Slug)
		}
	}
}

// TestMarksSortUpUnmarkedDown pins the full three-way order in one place.
func TestMarksSortUpUnmarkedDown(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	// Marked so that ID order and mark order disagree completely.
	if err := s.SetMark(ctx, "lru-cache", MarkUp, ""); err != nil { // #146
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "two-sum", MarkDown, ""); err != nil { // #1
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := s.Query(ctx, Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []string{"lru-cache", "trapping-rain-water", "two-sum"}
	for i, w := range want {
		if rows[i].Slug != w {
			t.Fatalf("board is %+v, want up / unmarked / down", rows)
		}
	}
}

func TestMarkMapCoversEveryVerdict(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.SetMark(ctx, "two-sum", MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := s.SetMark(ctx, "lru-cache", MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	got, err := s.MarkMap(ctx)
	if err != nil {
		t.Fatalf("MarkMap: %v", err)
	}
	if got["two-sum"] != MarkUp || got["lru-cache"] != MarkDown {
		t.Errorf("MarkMap = %+v", got)
	}
	if _, ok := got["never-marked"]; ok {
		t.Error("MarkMap invented an entry for an unmarked problem")
	}
}

// TestMarksSurviveAResync is the reason marks live in their own table: a re-sync rewrites
// the problems cache wholesale and must not take the user's judgments with it.
func TestMarksSurviveAResync(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("seed problems: %v", err)
	}
	if err := s.SetMark(ctx, "two-sum", MarkUp, "keep me"); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatalf("re-sync: %v", err)
	}

	got, err := s.MarkOf(ctx, "two-sum")
	if err != nil || got != MarkUp {
		t.Errorf("the mark did not survive a re-sync: %q (err %v)", got, err)
	}
}

// TestMarkNeedsNoSyncedProblem: an agent may mark something this machine has not pulled,
// and the verdict has to survive until it does. Same contract as the todo list (D-022).
func TestMarkNeedsNoSyncedProblem(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.SetMark(ctx, "some-problem-not-synced-yet", MarkUp, ""); err != nil {
		t.Fatalf("SetMark on an unsynced problem: %v", err)
	}
	all, _ := s.Marks(ctx, MarkNone)
	if len(all) != 1 {
		t.Fatalf("the mark was dropped: %+v", all)
	}
}
