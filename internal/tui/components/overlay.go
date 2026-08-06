package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Compositing one rendered block over another.
//
// Everything else in this package draws INTO a layout. A toast does not: it floats over
// whatever is already on screen, because the thing it interrupts is still what the user
// is looking at.
//
// The work is entirely about escape sequences. A rendered line is not a string of
// characters — it is characters interleaved with colour codes, so cutting it at "column
// 80" means cutting at the eightieth PRINTABLE cell while carrying the active styling
// across the seam. ansi.Truncate and ansi.TruncateLeft do exactly that, and they come
// from a module lipgloss already depends on, so this costs no new dependency.

// OverlayTopRight draws box over base, inset from the top and right edges.
//
// base is returned unchanged if the box cannot fit — a toast is never worth mangling the
// layout underneath it.
func OverlayTopRight(base, box string, top, rightInset int) string {
	lines := strings.Split(base, "\n")
	boxLines := strings.Split(strings.TrimRight(box, "\n"), "\n")
	if len(boxLines) == 0 || box == "" {
		return base
	}

	boxW := 0
	for _, l := range boxLines {
		if w := ansi.StringWidth(l); w > boxW {
			boxW = w
		}
	}

	// Width of the frame underneath, taken from its widest line: panes are padded to a
	// rectangle, so any full line gives the true width.
	baseW := 0
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > baseW {
			baseW = w
		}
	}

	left := baseW - rightInset - boxW
	if left < 0 || top < 0 || top+len(boxLines) > len(lines) {
		return base
	}

	for i, bl := range boxLines {
		row := top + i
		lines[row] = spliceLine(lines[row], bl, left, boxW, baseW)
	}
	return strings.Join(lines, "\n")
}

// spliceLine replaces the cells [left, left+boxW) of line with box.
//
// The tail is kept rather than dropped: on the right edge of the screen that tail is the
// pane's own border, and losing it would leave the frame visibly open for as long as the
// toast is up.
func spliceLine(line, box string, left, boxW, baseW int) string {
	// A short line (a blank spacer row, say) still has to accept the box, so pad first.
	if w := ansi.StringWidth(line); w < baseW {
		line += strings.Repeat(" ", baseW-w)
	}

	head := ansi.Truncate(line, left, "")
	if hw := ansi.StringWidth(head); hw < left {
		head += strings.Repeat(" ", left-hw)
	}

	tail := ansi.TruncateLeft(line, left+boxW, "")

	// Pad the box itself: its lines are ragged when the content is, and a ragged toast
	// would let the layout show through in the gaps.
	if bw := ansi.StringWidth(box); bw < boxW {
		box += strings.Repeat(" ", boxW-bw)
	}
	return head + box + tail
}
