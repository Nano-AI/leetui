package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	m = drive(t, m, key("tab")) // focus the problem pane
	if m.focus != paneDetail {
		t.Fatalf("focus = %v, want paneDetail", m.focus)
	}

	before := m.cursor
	m = drive(t, m, ctrl(tea.KeyCtrlD))
	if m.cursor != before {
		t.Errorf("ctrl-d moved the list cursor while the statement was focused")
	}
	if m.detailScroll == 0 {
		t.Error("ctrl-d did not scroll the statement")
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
