package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/store"
)

func ctrl(k tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: k} }

// TestHalfPageScroll covers ctrl-d / ctrl-u and their full-page siblings.
func TestHalfPageScroll(t *testing.T) {
	m := boot(t, true, 120, 34)
	half := m.visibleRows() / 2

	m = drive(t, m, ctrl(tea.KeyCtrlD))
	want := minInt(half, len(m.rows)-1)
	if m.cursor != want {
		t.Errorf("ctrl-d moved to %d, want %d", m.cursor, want)
	}

	m = drive(t, m, ctrl(tea.KeyCtrlU))
	if m.cursor != 0 {
		t.Errorf("ctrl-u returned to %d, want 0", m.cursor)
	}

	// Never past the ends.
	for i := 0; i < 20; i++ {
		m = drive(t, m, ctrl(tea.KeyCtrlD))
	}
	if m.cursor != len(m.rows)-1 {
		t.Errorf("cursor ran to %d, want the last row %d", m.cursor, len(m.rows)-1)
	}
	for i := 0; i < 20; i++ {
		m = drive(t, m, ctrl(tea.KeyCtrlU))
	}
	if m.cursor != 0 {
		t.Errorf("cursor ran to %d, want 0", m.cursor)
	}
}

// TestHalfPageScrollsStatementWhenDetailFocused: ctrl-d should do the expected thing
// wherever focus happens to be, rather than always driving the list.
func TestHalfPageScrollsStatementWhenDetailFocused(t *testing.T) {
	m := boot(t, true, 120, 34)
	// The statement lives on its own screen now (D-018), so open it first. tab on the
	// list is a no-op: there is only one pane there.
	m = drive(t, m, key("enter"))
	if m.mode != modeSolve {
		t.Fatalf("enter did not open the problem; mode = %v", m.mode)
	}
	if m.focus != paneDetail {
		t.Fatalf("focus = %v, want paneDetail", m.focus)
	}

	// Give it a statement long enough to scroll. The harness is offline, so nothing
	// fetches one, and with an empty body there is correctly nothing to move.
	m.detail = &store.Detail{Row: store.Row{Slug: m.rows[m.cursor].Slug}}
	m.detailMD = strings.Repeat("a line of the statement\n", 200)

	before := m.cursor
	m = drive(t, m, ctrl(tea.KeyCtrlD))
	if m.cursor != before {
		t.Errorf("ctrl-d moved the list cursor while the statement was focused")
	}
	if m.detailScroll == 0 {
		t.Error("ctrl-d did not scroll the statement")
	}
}

// TestStatementCannotScrollPastItsEnd is the reported bug: ctrl-d kept going after the
// last line, walking the statement off the top until one line sat in an empty pane.
// There is nothing below the end of a problem, so scrolling there is only ever a mistake
// to undo.
func TestStatementCannotScrollPastItsEnd(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	m.detail = &store.Detail{Row: store.Row{Slug: m.rows[m.cursor].Slug}}
	m.detailMD = strings.Repeat("a line of the statement\n", 40)

	max := m.detailMaxScroll()
	if max <= 0 {
		t.Fatalf("40 lines in a 34-row terminal should be scrollable; max = %d", max)
	}

	// Far more half-pages than the statement has.
	for i := 0; i < 40; i++ {
		m = drive(t, m, ctrl(tea.KeyCtrlD))
	}
	if m.detailScroll > max {
		t.Errorf("scrolled to %d, past the last screenful at %d", m.detailScroll, max)
	}

	// And the last screenful is still FULL — stopping one line from the bottom would
	// leave the pane almost empty, which is the same failure by a smaller margin.
	if m.detailScroll != max {
		t.Errorf("scroll settled at %d, want %d", m.detailScroll, max)
	}

	// G goes to the same place, and g comes back.
	m = drive(t, m, key("G"))
	if m.detailScroll != max {
		t.Errorf("G left the scroll at %d, want %d", m.detailScroll, max)
	}
	m = drive(t, m, key("g"))
	if m.detailScroll != 0 {
		t.Errorf("g left the scroll at %d, want 0", m.detailScroll)
	}
}

// TestShortStatementDoesNotScroll: a problem that already fits has nowhere to go, and
// ctrl-d must be a no-op rather than blanking the pane.
func TestShortStatementDoesNotScroll(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	m.detail = &store.Detail{Row: store.Row{Slug: m.rows[m.cursor].Slug}}
	m.detailMD = "one line\ntwo lines\n"

	m = drive(t, m, ctrl(tea.KeyCtrlD), ctrl(tea.KeyCtrlD))
	if m.detailScroll != 0 {
		t.Errorf("a statement that fits scrolled to %d", m.detailScroll)
	}
}

// TestDetailHeaderDoesNotBlankOnMove is the jitter guard.
//
// The heading is rendered from the board row, which is always in memory, so moving the
// cursor must repaint it immediately — never blank the pane and repaint a moment later.
func TestDetailHeaderDoesNotBlankOnMove(t *testing.T) {
	m := boot(t, true, 120, 34)

	for i := 0; i < 3; i++ {
		m = drive(t, m, key("j"))
		want := m.rows[m.cursor].Title
		if got := m.viewDetail(70, 14); !containsPlain(got, want) {
			t.Fatalf("after %d moves the problem pane does not show %q:\n%s", i+1, want, got)
		}
	}
}

func containsPlain(s, want string) bool {
	return len(want) > 0 && len(s) > 0 && indexOf(stripANSI(s), want) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
