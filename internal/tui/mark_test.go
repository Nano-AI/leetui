package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// TestMarkKeysSetAndToggle: `+` and `-` record a verdict, and pressing the same key again
// takes it back.
func TestMarkKeysSetAndToggle(t *testing.T) {
	m := boot(t, true, 120, 32)
	slug := m.rows[0].Slug

	m = drive(t, m, key("+"))
	if m.marks[slug] != store.MarkUp {
		t.Fatalf("+ left the mark as %q, want up", m.marks[slug])
	}
	// It reached SQLite, not just the in-memory map.
	if got, _ := m.store.MarkOf(context.Background(), slug); got != store.MarkUp {
		t.Errorf("the mark was not persisted: %q", got)
	}

	m = drive(t, m, key("+"))
	if _, ok := m.marks[slug]; ok {
		t.Errorf("pressing + twice left %q marked; it should clear", slug)
	}
	if got, _ := m.store.MarkOf(context.Background(), slug); got != store.MarkNone {
		t.Errorf("the clear was not persisted: %q", got)
	}
}

// TestMarkKeysFlipDirectly: - on an important row makes it unimportant in one press,
// rather than needing a clear first.
func TestMarkKeysFlipDirectly(t *testing.T) {
	m := boot(t, true, 120, 32)
	slug := m.rows[0].Slug

	m = drive(t, m, key("+"))
	m = drive(t, m, key("-"))
	if m.marks[slug] != store.MarkDown {
		t.Fatalf("- on an important row gave %q, want down", m.marks[slug])
	}
	if got, _ := m.store.MarkOf(context.Background(), slug); got != store.MarkDown {
		t.Errorf("the flip was not persisted: %q", got)
	}
}

// TestMarkColumnShowsTheVerdict covers the board rendering and its header — a glyph is
// only allowed to be a glyph when something else names it (D-023).
func TestMarkColumnShowsTheVerdict(t *testing.T) {
	m := boot(t, true, 120, 32)

	out := m.View()
	if !strings.Contains(strings.ToUpper(out), "MARK") {
		t.Fatal("the board has no MARK header")
	}

	m = drive(t, m, key("+"))
	if !strings.Contains(m.View(), "+") {
		t.Error("an important row does not draw its glyph")
	}
}

// TestMarkFilterCycles is the re-do loop: narrow to the important ones, then the written
// off ones, then back to everything.
func TestMarkFilterCycles(t *testing.T) {
	m := boot(t, true, 120, 32)
	ctx := context.Background()
	if err := m.store.SetMark(ctx, "two-sum", store.MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	if err := m.store.SetMark(ctx, "lru-cache", store.MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	m = drive(t, m, key("i"))
	if m.filter.Mark != store.MarkUp {
		t.Fatalf("first i gave %q, want up", m.filter.Mark)
	}
	if len(m.rows) != 1 || m.rows[0].Slug != "two-sum" {
		t.Fatalf("important-only shows %+v, want just two-sum", m.rows)
	}
	// The bezel uppercases what it is given, so compare case-insensitively.
	if out := strings.ToUpper(m.View()); !strings.Contains(out, "IMPORTANT") {
		t.Error("the active mark filter is not named on the bezel")
	}

	m = drive(t, m, key("i"))
	if m.filter.Mark != store.MarkDown {
		t.Fatalf("second i gave %q, want down", m.filter.Mark)
	}
	if len(m.rows) != 1 || m.rows[0].Slug != "lru-cache" {
		t.Fatalf("unimportant-only shows %+v, want just lru-cache", m.rows)
	}

	m = drive(t, m, key("i"))
	if m.filter.Mark != store.MarkNone {
		t.Errorf("third i left the filter as %q, want cleared", m.filter.Mark)
	}
	if len(m.rows) != 4 {
		t.Errorf("clearing the filter left %d rows, want all 4", len(m.rows))
	}
}

// TestMarkFilterDropsARowThatNoLongerMatches: clearing a verdict while filtered to it
// must remove the row, or the board contradicts its own bezel.
func TestMarkFilterDropsARowThatNoLongerMatches(t *testing.T) {
	m := boot(t, true, 120, 32)
	ctx := context.Background()
	if err := m.store.SetMark(ctx, "two-sum", store.MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	m = drive(t, m, key("i"))
	if len(m.rows) != 1 {
		t.Fatalf("important-only shows %d rows, want 1", len(m.rows))
	}

	// Un-mark the row under the cursor while the filter is showing.
	m = drive(t, m, key("+"))
	if len(m.rows) != 0 {
		t.Errorf("the un-marked row is still on an important-only board: %+v", m.rows)
	}
}

// TestUnimportantSinksAndGreys is the behaviour `-` is for (D-032a): the row stays on the
// board, moves to the bottom, and stops competing for attention.
func TestUnimportantSinksAndGreys(t *testing.T) {
	m := boot(t, true, 120, 32)
	ctx := context.Background()
	// two-sum is #1 and leads the default board.
	if err := m.store.SetMark(ctx, "two-sum", store.MarkDown, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}
	// Three presses of i cycle the filter up → down → off, and each one reloads the rows,
	// which is how a mark written behind the model's back reaches the board.
	m = drive(t, m, key("i"), key("i"), key("i"))
	if m.filter.Mark != store.MarkNone {
		t.Fatalf("the filter did not cycle back off: %q", m.filter.Mark)
	}

	if len(m.rows) != 4 {
		t.Fatalf("the unimportant problem left the board: %d rows, want 4", len(m.rows))
	}
	if m.rows[len(m.rows)-1].Slug != "two-sum" {
		t.Errorf("last row is %q, want two-sum sunk to the bottom", m.rows[len(m.rows)-1].Slug)
	}

	// It is still on screen, which is the whole difference from removing it.
	if !strings.Contains(m.View(), "Two Sum") {
		t.Error("the demoted problem is not on screen at all")
	}
}

// TestDimmedRowDropsItsColour: greying is what stops a demoted row pulling the eye. The
// difficulty tag is the loudest thing in a row, so it is the one worth asserting on.
func TestDimmedRowDropsItsColour(t *testing.T) {
	m := boot(t, true, 120, 32)
	r := m.rows[0]

	bright := m.viewProblemRow(r, 1, false, boardLayout(100), nil)

	m.marks = map[string]store.Mark{r.Slug: store.MarkDown}
	dimmed := m.viewProblemRow(r, 1, false, boardLayout(100), nil)

	if bright == dimmed {
		t.Fatal("a row marked unimportant renders identically to one that is not")
	}

	// Compare against the theme's own rendering rather than a colour literal: lipgloss
	// emits truecolor as "38;2;28;186;186", not as the hex the palette is written in.
	coloured := theme.Easy.Render()
	if !strings.Contains(bright, coloured) {
		t.Fatalf("the undimmed row is missing its coloured difficulty tag")
	}
	if strings.Contains(dimmed, coloured) {
		t.Error("a dimmed row kept its difficulty colour")
	}
	if !strings.Contains(dimmed, theme.Meta.Render("ESY")) {
		t.Error("dimming dropped the difficulty tag rather than just draining its colour")
	}
}

// TestCursorBeatsDimming: a row you have deliberately moved onto must be readable,
// whatever you decided about it earlier.
func TestCursorBeatsDimming(t *testing.T) {
	m := boot(t, true, 120, 32)
	r := m.rows[0]
	m.marks = map[string]store.Mark{r.Slug: store.MarkDown}

	unselected := m.viewProblemRow(r, 1, false, boardLayout(100), nil)
	selected := m.viewProblemRow(r, 1, true, boardLayout(100), nil)

	if unselected == selected {
		t.Error("the cursor row renders the same as a dimmed one; selection must win")
	}
}

// TestMarkAndTodoAreSeparate is the premise of D-032: a mark is not a todo, and marking
// must not touch the list.
func TestMarkAndTodoAreSeparate(t *testing.T) {
	m := boot(t, true, 120, 32)
	slug := m.rows[0].Slug

	m = drive(t, m, key("+"))
	if m.todo[slug] {
		t.Error("marking a problem important put it on the todo list")
	}

	m = drive(t, m, key("m"))
	if !m.todo[slug] {
		t.Fatal("m did not add a todo")
	}
	// Both can hold at once, and neither clears the other.
	if m.marks[slug] != store.MarkUp {
		t.Error("adding a todo cleared the mark")
	}
}

// TestMarkSurvivesSolved is the re-do case the todo list cannot express. two-sum is
// accepted in the harness's seed data.
func TestMarkSurvivesSolved(t *testing.T) {
	m := boot(t, true, 120, 32)
	ctx := context.Background()
	if err := m.store.SetMark(ctx, "two-sum", store.MarkUp, ""); err != nil {
		t.Fatalf("SetMark: %v", err)
	}

	rows, err := m.store.Query(ctx, store.Filter{Mark: store.MarkUp, Status: "ac"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0].Slug != "two-sum" {
		t.Fatalf("solved-and-important gave %+v, want two-sum", rows)
	}
}
