package tui

import (
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestFramesFitTerminal is the guard that matters most: a frame wider than the terminal
// turns the whole TUI into garbage, and it is invisible in unit tests of anything else.
func TestFramesFitTerminal(t *testing.T) {
	cases := []struct {
		name   string
		w, h   int
		seeded bool
		keys   []tea.Msg
	}{
		{"wide", 120, 32, true, nil},
		{"mid", 92, 28, true, nil},
		{"narrow", 78, 24, true, nil},
		{"tiny", 60, 16, true, nil},
		{"empty store", 120, 32, false, nil},
		{"help", 120, 32, true, []tea.Msg{key("?")}},
		{"auth", 120, 32, true, []tea.Msg{key("a")}},
		{"searching", 120, 32, true, []tea.Msg{key("/"), key("l")}},
		{"filtered", 120, 32, true, []tea.Msg{key("2")}},
		{"detail focus", 120, 32, true, []tea.Msg{key("tab")}},
		{"git", 120, 32, true, []tea.Msg{key("v")}},
		{"git narrow", 78, 24, true, []tea.Msg{key("v")}},

		// 80x24 is the floor every terminal emulator still defaults to, so it gets
		// every screen rather than a sample.
		{"80x24 board", 80, 24, true, nil},
		{"80x24 solve", 80, 24, true, []tea.Msg{key("enter")}},
		{"80x24 help", 80, 24, true, []tea.Msg{key("?")}},
		{"80x24 settings", 80, 24, true, []tea.Msg{key("V")}},
		{"80x24 git", 80, 24, true, []tea.Msg{key("v")}},
		{"80x24 palette", 80, 24, true, []tea.Msg{key(":")}},
		{"80x24 search", 80, 24, true, []tea.Msg{key("/"), key("l")}},
		{"80x24 companies", 80, 24, true, []tea.Msg{key("c")}},
		{"80x24 auth", 80, 24, true, []tea.Msg{key("a")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := boot(t, tc.seeded, tc.w, tc.h)
			m = drive(t, m, tc.keys...)

			out := m.View()
			t.Logf("\n%s", out)

			lines := strings.Split(out, "\n")
			for i, line := range lines {
				if got := lipgloss.Width(line); got > tc.w {
					t.Errorf("line %d overflows: %d > %d cols", i, got, tc.w)
				}
			}
			if len(lines) > tc.h+1 {
				t.Errorf("frame is %d lines, exceeds terminal height %d", len(lines), tc.h)
			}
		})
	}
}

func TestBoardShowsSeededProblems(t *testing.T) {
	m := boot(t, true, 120, 32)

	if len(m.rows) != 4 {
		t.Fatalf("loaded %d rows, want 4", len(m.rows))
	}
	out := m.View()
	for _, want := range []string{"Two Sum", "LRU Cache", "Trapping Rain Water"} {
		if !strings.Contains(out, want) {
			t.Errorf("board is missing %q", want)
		}
	}
	// Row state is a glyph under a header that names the column (D-023). The header is
	// what makes the marks legible; without it this is the unlabelled-glyph mistake.
	if !strings.Contains(stripANSI(out), "STATE") {
		t.Error("the state column lost its header")
	}
	g := theme.Glyphs()
	for name, want := range map[string]string{
		"solved": g.Solved, "tried": g.Tried, "locked": g.Locked,
	} {
		if !strings.Contains(stripANSI(out), want) {
			t.Errorf("board is missing the %s mark %q", name, want)
		}
	}
}

// TestBoardIsAClosedGrid guards the frame: every content line must begin and end with a
// bezel character. A row that escapes the frame is the failure the borders exist to
// prevent, and it is invisible in a width-only check.
func TestBoardIsAClosedGrid(t *testing.T) {
	m := boot(t, true, 120, 32)
	board := m.viewBoard(120, 16)

	lines := strings.Split(strings.TrimRight(board, "\n"), "\n")
	if len(lines) != 16 {
		t.Fatalf("board rendered %d lines, want exactly 16", len(lines))
	}

	for i, line := range lines {
		plain := []rune(stripANSI(line))
		if len(plain) == 0 {
			t.Errorf("line %d is empty", i)
			continue
		}
		first, last := plain[0], plain[len(plain)-1]
		if !strings.ContainsRune("╭│├╰", first) || !strings.ContainsRune("╮│┤╯", last) {
			t.Errorf("line %d is not enclosed: starts %q ends %q", i, first, last)
		}
		if lipgloss.Width(line) != 120 {
			t.Errorf("line %d is %d cols, want 120", i, lipgloss.Width(line))
		}
	}
}
