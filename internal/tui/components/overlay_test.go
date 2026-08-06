package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func base(w, h int) string {
	lines := make([]string, h)
	for i := range lines {
		lines[i] = strings.Repeat(".", w)
	}
	return strings.Join(lines, "\n")
}

func TestOverlayPlacesTheBoxFlushRight(t *testing.T) {
	got := OverlayTopRight(base(20, 5), "ABCD\nEFGH", 1, 0)
	lines := strings.Split(got, "\n")

	if lines[0] != strings.Repeat(".", 20) {
		t.Errorf("row 0 was disturbed: %q", lines[0])
	}
	if want := strings.Repeat(".", 16) + "ABCD"; lines[1] != want {
		t.Errorf("row 1 = %q, want %q", lines[1], want)
	}
	if want := strings.Repeat(".", 16) + "EFGH"; lines[2] != want {
		t.Errorf("row 2 = %q, want %q", lines[2], want)
	}
	if lines[3] != strings.Repeat(".", 20) {
		t.Errorf("row 3 was disturbed: %q", lines[3])
	}
}

// TestOverlayKeepsTheTail is why an inset is usable at all: on the right edge that tail is
// the pane's own border, and dropping it would leave the frame visibly open.
func TestOverlayKeepsTheTail(t *testing.T) {
	got := OverlayTopRight(base(20, 3), "XX", 1, 3)
	lines := strings.Split(got, "\n")

	if want := strings.Repeat(".", 15) + "XX" + "..."; lines[1] != want {
		t.Errorf("row 1 = %q, want %q", lines[1], want)
	}
}

// TestOverlayCountsCellsNotBytes is the whole reason this is not a substring operation:
// a rendered line is characters interleaved with colour codes.
func TestOverlayCountsCellsNotBytes(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(strings.Repeat(".", 20))
	if w := ansi.StringWidth(styled); w != 20 {
		t.Fatalf("fixture is %d cells wide, want 20", w)
	}

	got := OverlayTopRight(styled+"\n"+styled+"\n"+styled, "XX", 1, 0)
	for i, line := range strings.Split(got, "\n") {
		if w := ansi.StringWidth(line); w != 20 {
			t.Errorf("row %d is %d cells after overlay, want 20: %q", i, w, line)
		}
	}
	if !strings.Contains(stripSGR(strings.Split(got, "\n")[1]), "XX") {
		t.Error("the box did not land on the styled row")
	}
}

// TestOverlayRefusesToMangle: a toast is never worth breaking the layout under it.
func TestOverlayRefusesToMangle(t *testing.T) {
	b := base(10, 3)

	if got := OverlayTopRight(b, strings.Repeat("X", 40), 1, 0); got != b {
		t.Error("a box wider than the frame was drawn anyway")
	}
	if got := OverlayTopRight(b, "X\nX\nX\nX\nX", 1, 0); got != b {
		t.Error("a box taller than the remaining rows was drawn anyway")
	}
	if got := OverlayTopRight(b, "", 1, 0); got != b {
		t.Error("an empty box changed the frame")
	}
}

// TestOverlayPadsShortRows: a blank spacer row is shorter than the frame, and the box
// still has to land in the right column on it.
func TestOverlayPadsShortRows(t *testing.T) {
	b := strings.Repeat(".", 20) + "\n" + "" + "\n" + strings.Repeat(".", 20)

	got := OverlayTopRight(b, "XX", 1, 0)
	line := strings.Split(got, "\n")[1]
	if ansi.StringWidth(line) != 20 {
		t.Errorf("short row is %d cells after overlay: %q", ansi.StringWidth(line), line)
	}
	if !strings.HasSuffix(line, "XX") {
		t.Errorf("box did not land flush right on a short row: %q", line)
	}
}

func stripSGR(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || r == 'K'):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
