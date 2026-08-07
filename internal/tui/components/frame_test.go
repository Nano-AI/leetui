package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestOneBodyLineIsOneFrameRow is the regression that reached a user's screen: a
// statement's code block is not wrapped by Glamour, so a long example line was handed to
// the frame intact. lipgloss's Width() wrapped it, the row became two rows, the right
// bezel landed after the second, and every row below was shifted — the border appeared to
// shear off halfway down the pane.
func TestOneBodyLineIsOneFrameRow(t *testing.T) {
	const w, h = 40, 8

	long := "[4, 5, 0, -2, -3, 1], [5], [5, 0], [5, 0, -2, -3], [0], [0, -2, -3], [-2, -3]"
	f := Frame{Title: "problem", Width: w, Height: h}

	out := f.Render(" " + long)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) != h {
		t.Fatalf("frame is %d lines, want exactly %d", len(lines), h)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != w {
			t.Errorf("line %d is %d cells, want %d:\n%q", i, got, w, line)
		}
	}
}

// TestEveryRowIsExactlyWidth covers the padding half: a SHORT line has to be filled out
// to the bezel, or the right edge staggers row to row.
func TestEveryRowIsExactlyWidth(t *testing.T) {
	for _, w := range []int{20, 40, 80, 120} {
		f := Frame{Title: "run", Right: "3 passed", Width: w, Height: 6}
		out := f.Render(" short\n\n " + strings.Repeat("x", w*2))

		for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if got := ansi.StringWidth(line); got != w {
				t.Errorf("width %d, line %d is %d cells:\n%q", w, i, got, line)
			}
		}
	}
}

// TestTruncationKeepsStyling guards the other way this can break: truncating a styled
// line must not leave a dangling escape sequence that bleeds colour into the bezel.
func TestTruncationKeepsStyling(t *testing.T) {
	styled := "\x1b[38;2;232;163;60m" + strings.Repeat("amber ", 40) + "\x1b[0m"
	f := Frame{Title: "t", Width: 30, Height: 4}
	out := f.Render(styled)

	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "\x1b[0m") &&
		!strings.Contains(out, "\x1b[0m") {
		t.Error("styled content was truncated without any reset — colour will bleed")
	}
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if got := ansi.StringWidth(line); got != 30 {
			t.Errorf("line %d is %d cells, want 30", i, got)
		}
	}
}
