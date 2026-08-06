package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// bg is the escape lipgloss emits for a background colour, so a test can look for a
// specific band rather than for "some colour".
func bg(c lipgloss.TerminalColor) string {
	const probe = "\x00"
	r := lipgloss.NewStyle().Background(c).Render(probe)
	return r[:strings.Index(r, probe)]
}

// boardRows returns the rendered rows that carry a problem.
//
// Matched on the zero-padded ID rather than the title: the ID column is a fixed width and
// is never truncated, whereas a long title is — which made this helper silently miss rows
// the moment a new column narrowed the title.
func boardRows(t *testing.T, m Model) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(m.viewBoard(100, m.boardHeight()), "\n") {
		plain := stripANSI(line)
		for _, r := range m.rows {
			if strings.Contains(plain, fmt.Sprintf("%04d", r.NumericID)) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

// TestRowsAreBanded is the fix for "I can't trace along and find the right one". Vertical
// rules separate the columns but do nothing to carry the eye ACROSS a row.
func TestRowsAreBanded(t *testing.T) {
	m := boot(t, true, 100, 20)
	rows := boardRows(t, m)
	if len(rows) < 4 {
		t.Fatalf("found %d problem rows, want at least 4", len(rows))
	}

	band := bg(theme.Band)
	if strings.Contains(rows[0], band) {
		t.Error("the first row is banded; banding should start on the second")
	}
	if !strings.Contains(rows[1], band) {
		t.Error("the second row is not banded")
	}
	if strings.Contains(rows[2], band) {
		t.Error("the third row is banded; bands must alternate")
	}
	if !strings.Contains(rows[3], band) {
		t.Error("the fourth row is not banded")
	}
}

// TestCursorRowOutranksTheBand: the selected row must be unmistakable from across the
// screen, so it takes its own colour regardless of which band it fell on.
func TestCursorRowOutranksTheBand(t *testing.T) {
	m := boot(t, true, 100, 20)

	// Row 1 is a banded row; putting the cursor there is the case that would collide.
	m = drive(t, m, key("down"))
	if m.cursor != 1 {
		t.Fatalf("cursor is at %d, want 1", m.cursor)
	}

	rows := boardRows(t, m)
	if !strings.Contains(rows[1], bg(theme.Cursor)) {
		t.Error("the cursor row does not carry the cursor colour")
	}
	if strings.Contains(rows[1], bg(theme.Band)) {
		t.Error("the cursor row is still wearing its band")
	}
}

// TestBandCoversTheWholeRow is what makes banding work at all: a band that stops at the
// first styled cell would be worse than none, because the eye would follow it and lose
// the line halfway.
func TestBandCoversTheWholeRow(t *testing.T) {
	m := boot(t, true, 100, 20)
	row := boardRows(t, m)[1]

	// lipgloss closes every styled cell with a reset. Each one must be followed by the
	// band being re-asserted, or the row goes transparent from that point on.
	band := bg(theme.Band)
	segs := strings.Split(row, "\x1b[0m")

	painted := 0
	for i, seg := range segs {
		// Skip the first segment (it precedes any reset) and the frame's own borders,
		// which are drawn AROUND the row rather than as part of it. Identify those by
		// what they contain rather than by position — the border is second to last, with
		// an empty segment after it.
		text := stripANSI(seg)
		if i == 0 || text == "" || strings.Trim(text, "│╭╮╰╯ ") == "" && !strings.Contains(seg, band) {
			continue
		}
		if !strings.HasPrefix(seg, band) {
			t.Errorf("segment %d does not re-assert the band: %q", i, seg)
			return
		}
		painted++
	}

	if painted < 4 {
		t.Errorf("only %d segments carry the band; the row has more cells than that", painted)
	}

	// The decisive one: the padding after the LAST cell must still be banded, or the row
	// visibly gives out before it reaches the right-hand border.
	last := segs[len(segs)-2]
	if strings.Contains(stripANSI(last), "│") {
		last = segs[len(segs)-3]
	}
	if !strings.Contains(last, band) {
		t.Errorf("the band stops before the end of the row: %q", last)
	}
}

// TestLockedDependsOnTheAccount: with a subscription nothing is locked, and the column
// claims to report the reader's state rather than a property of the problem.
func TestLockedDependsOnTheAccount(t *testing.T) {
	m := boot(t, true, 100, 20)

	if !strings.Contains(stripANSI(m.View()), "LOCKED") {
		t.Fatal("a signed-out account sees no LOCKED; the fixture has a paid-only problem")
	}

	m.premium = true
	if strings.Contains(stripANSI(m.View()), "LOCKED") {
		t.Error("a Premium account is still being told a problem is locked")
	}
}

// TestPremiumStatementIsNotGated: the detail pane read off PaidOnly too, so a subscriber
// opening a paid problem was shown a dead end instead of a statement on its way.
func TestPremiumStatementIsNotGated(t *testing.T) {
	m := boot(t, true, 100, 20)
	m.cursor = 3 // the paid-only fixture
	m = drive(t, m, key("enter"))

	if !strings.Contains(stripANSI(m.View()), "is Premium") {
		t.Fatal("a signed-out account is not shown the gate")
	}

	m.premium = true
	if strings.Contains(stripANSI(m.View()), "is Premium") {
		t.Error("a Premium account is still shown the gate")
	}
}
