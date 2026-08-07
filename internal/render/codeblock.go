package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Making an example block look like a block.
//
// The examples are the part of a statement you check yourself against — the input, the
// output, the walk-through — and they arrived in a grey one shade off body text with no
// background at all. On a wall of paragraphs that reads as more paragraph, which is
// exactly the complaint: you cannot see where an example starts or stops.
//
// The style now gives them the flap background that inline `code` already carries. But
// Glamour colours only the CHARACTERS and pads the rest of each line with plain spaces,
// so the background stops at the last character and the block comes out ragged. This
// pass fills those lines out, which is what turns a coloured run of text into a bounded
// object.

// paintCodeBlocks extends a code block's background across the full width of its lines.
//
// A code-block line is one whose FIRST escape sequence sets a background: Glamour emits
// the block's style before any content. Inline `code` also sets a background, but it
// appears mid-line after the paragraph's own foreground, so the two do not collide.
//
// The fill reuses the sequence found in the line rather than generating one. Glamour
// renders through its own colour profile, which is not always the one lipgloss would
// pick — matching a self-generated truecolor sequence against Glamour's 256-colour
// output finds nothing, which is exactly the bug the first attempt had.
func paintCodeBlocks(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		bg, ok := leadingBackground(line)
		if !ok {
			continue
		}
		lines[i] = fillLine(line, bg, width)
	}
	return strings.Join(lines, "\n")
}

// leadingBackground returns the escape run at the start of the line when it sets a
// background, which marks the line as code-block content.
//
// On a terminal with no colour there is no such sequence anywhere, no line matches, and
// the whole pass becomes a no-op rather than a special case.
func leadingBackground(line string) (string, bool) {
	if !strings.HasPrefix(line, esc) {
		return "", false
	}

	// Consume the whole run of escapes at the head of the line — Glamour emits the
	// foreground and background as separate sequences.
	end := 0
	for strings.HasPrefix(line[end:], esc) {
		m := strings.IndexByte(line[end:], 'm')
		if m < 0 {
			return "", false
		}
		end += m + 1
	}

	head := line[:end]
	if !strings.Contains(head, "48;") {
		return "", false
	}
	return head, true
}

// fillLine rebuilds one code-block line as content, then background out to width.
//
// Glamour's own trailing padding is dropped FIRST, always — including when the line
// already reaches the width. That case looked safe and was the bug: Glamour indents a
// code block by its margin, so a full line plus that padding is wider than the pane,
// and the pane's wrap pushed the overflowing spaces onto a row of their own. A blank
// band under an example, from padding nobody could see.
//
// Dropping the padding also removes a second problem: it is unstyled, so left in place
// it sits as a bar of page colour between the text and the fill.
func fillLine(line, bg string, width int) string {
	visible := ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " "))
	content := ansi.Truncate(line, visible, "")
	if visible >= width {
		return content
	}
	return content + bg + strings.Repeat(" ", width-visible) + reset
}

const (
	esc   = "\x1b["
	reset = "\x1b[0m"
)
