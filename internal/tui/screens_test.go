package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// TestBrowseScreenHasNoStatement is the requirement: while you are looking FOR a problem,
// one you have not chosen to read is in the way, and it costs the list 40% of the width it
// wants for titles and tags (D-018).
func TestBrowseScreenHasNoStatement(t *testing.T) {
	m := boot(t, true, 120, 30)
	out := stripANSI(m.View())

	for _, gone := range []string{"PROBLEM ─", "SOLUTION ─", "SUBMISSIONS"} {
		if strings.Contains(out, gone) {
			t.Errorf("the browse screen still shows %q:\n%s", gone, out)
		}
	}
	if !strings.Contains(out, "PROBLEMS") {
		t.Error("the list is not on the browse screen")
	}
}

// TestBrowseListUsesTheFullWidth: the whole point of dropping the statement is that the
// titles get the room.
func TestBrowseListUsesTheFullWidth(t *testing.T) {
	m := boot(t, true, 120, 30)

	long := "Lowest Common Ancestor of a Binary Tree III"
	if !strings.Contains(stripANSI(m.View()), long) {
		t.Errorf("a long title is being truncated on a 120-column browse screen")
	}

	// And the list should fill the body rather than stopping half way.
	lines := strings.Split(strings.TrimRight(stripANSI(m.viewBoard(120, m.boardHeight())), "\n"), "\n")
	if len(lines) < 20 {
		t.Errorf("the list is only %d lines tall on a 30-row terminal", len(lines))
	}
}

func TestEnterOpensAndEscReturns(t *testing.T) {
	m := boot(t, true, 120, 30)
	m = drive(t, m, key("down")) // sit on row 1, not row 0

	at := m.cursor
	m = drive(t, m, key("enter"))
	if m.mode != modeSolve {
		t.Fatalf("enter did not open the problem; mode = %v", m.mode)
	}
	if !strings.Contains(stripANSI(m.View()), "SOLUTION") {
		t.Error("the solve screen is missing the working column")
	}

	m = drive(t, m, key("esc"))
	if m.mode != modeBoard {
		t.Errorf("esc did not return to the list; mode = %v", m.mode)
	}
	// Coming back must land where you left, or browsing becomes a game of finding your
	// place again.
	if m.cursor != at {
		t.Errorf("cursor moved to %d while opening a problem; was %d", m.cursor, at)
	}
}

// TestEscPeelsOneLayer: the editorial is inside the problem screen, so esc closes that
// first rather than jumping straight back to the list.
func TestEscPeelsOneLayer(t *testing.T) {
	m := boot(t, true, 120, 30)
	m = drive(t, m, key("enter"), key("d"))
	if !m.showEditorial {
		t.Fatal("d did not open the editorial")
	}

	m = drive(t, m, key("esc"))
	if m.showEditorial {
		t.Error("esc did not close the editorial")
	}
	if m.mode != modeSolve {
		t.Errorf("esc left the problem screen as well; mode = %v", m.mode)
	}

	m = drive(t, m, key("esc"))
	if m.mode != modeBoard {
		t.Errorf("the second esc did not return to the list; mode = %v", m.mode)
	}
}

// TestTabDoesNothingOnTheList: one pane means focus has nowhere to go, and moving it
// somewhere invisible would make the next keypress behave unexpectedly.
func TestTabDoesNothingOnTheList(t *testing.T) {
	m := boot(t, true, 120, 30)
	before := m.focus

	m = drive(t, m, key("tab"))
	if m.focus != before {
		t.Errorf("tab moved focus to %v on a single-pane screen", m.focus)
	}
}

// TestNumbersFollowTheScreen: on the list they filter by difficulty, on a problem they
// open markers. The screen decides, which is plainer than focus deciding.
func TestNumbersFollowTheScreen(t *testing.T) {
	m := boot(t, true, 120, 30)

	m = drive(t, m, key("2"))
	if len(m.filter.Difficulty) != 1 || m.filter.Difficulty[0] != "Medium" {
		t.Fatalf("2 on the list did not filter to Medium: %v", m.filter.Difficulty)
	}

	m = drive(t, m, key("0"), key("enter"))
	before := m.filter.Difficulty
	m = drive(t, m, key("2"))
	if len(m.filter.Difficulty) != len(before) {
		t.Errorf("2 on the problem screen changed the list filter to %v", m.filter.Difficulty)
	}
}

// TestHintsFollowTheScreen: offering "r run" while scrolling four thousand problems
// pushes the one key that matters off the end of the strip.
func TestHintsFollowTheScreen(t *testing.T) {
	m := boot(t, true, 120, 30)

	browse := strings.Join(m.hintKeys(), " ")
	if !strings.Contains(browse, "enter open") {
		t.Errorf("the browse hints do not offer enter: %q", browse)
	}
	if strings.Contains(browse, "r run") {
		t.Errorf("the browse hints offer run, which needs a problem open: %q", browse)
	}

	m = drive(t, m, key("enter"))
	solve := strings.Join(m.hintKeys(), " ")
	for _, want := range []string{"r run", "s submit", "esc back"} {
		if !strings.Contains(solve, want) {
			t.Errorf("the solve hints are missing %q: %q", want, solve)
		}
	}
}

// TestDifficultyUsesLeetCodesColours — the borrowed palette is the point: anyone who has
// used LeetCode reads teal/amber/red without a legend (D-017).
func TestDifficultyUsesLeetCodesColours(t *testing.T) {
	for _, tc := range []struct {
		d    theme.Difficulty
		hex  string
		name string
	}{
		{theme.Easy, "#1CBABA", "easy teal"},
		{theme.Medium, "#FFB700", "medium amber"},
		{theme.Hard, "#F63737", "hard red"},
	} {
		if got := string(tc.d.Color()); !strings.EqualFold(got, tc.hex) {
			t.Errorf("%s = %s, want %s", tc.name, got, tc.hex)
		}
	}

	// And they reach the board, not just the palette.
	m := boot(t, true, 120, 30)
	out := m.View()
	for _, rgb := range []string{"28;186;186", "255;183;0", "246;55;55"} {
		if !strings.Contains(out, rgb) {
			t.Errorf("the board never renders rgb(%s)", rgb)
		}
	}
}
