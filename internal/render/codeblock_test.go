package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// An example block is the part of a statement you check yourself against. Glamour colours
// its characters and pads the rest of the line with plain spaces, so the background used
// to stop at the last character and the block read as a ragged smear — or, before the
// background existed at all, as more paragraph.
func TestExampleBlocksFillTheirWidth(t *testing.T) {
	const width = 74

	md := "Given two arrays.\n\n**Example 1:**\n\n```\nInput: gas = [1,2,3,4,5]\nOutput: 3\nExplanation: a much longer line of walk-through text that keeps going\n```\n\n**Constraints:**\n\n- `n == gas.length`\n"

	out, err := Markdown(md, width)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var blockLines int
	for i, line := range strings.Split(out, "\n") {
		bg, ok := leadingBackground(line)
		if !ok {
			continue
		}
		blockLines++
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("line %d is %d cells, want %d — the block will look ragged:\n%q",
				i, got, width, line)
		}
		if bg == "" {
			t.Errorf("line %d matched with an empty background sequence", i)
		}
	}

	if blockLines == 0 {
		t.Fatal("no code-block lines were detected; the examples have no background at all")
	}
}

// TestProseIsNotPainted is the discriminator. Inline `code` also sets a background, and
// painting its whole line would put a bar across the middle of a paragraph.
func TestProseIsNotPainted(t *testing.T) {
	out, err := Markdown("A paragraph mentioning `nums` inline, and nothing else.\n", 60)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for i, line := range strings.Split(out, "\n") {
		if _, ok := leadingBackground(line); ok {
			t.Errorf("prose line %d was treated as a code block:\n%q", i, line)
		}
	}
}

// TestNoCodeBlockLineExceedsWidth is the blank-band bug: Glamour indents a code block by
// its margin, so a full line plus its trailing padding came out WIDER than the pane. The
// pane's wrap then pushed those invisible spaces onto a row of their own, leaving a band
// of background under the example with nothing in it.
func TestNoCodeBlockLineExceedsWidth(t *testing.T) {
	for _, width := range []int{40, 58, 74, 100} {
		// A line engineered to land exactly at the limit, plus one comfortably over.
		long := strings.Repeat("x", width)
		md := "Text.\n\n```\n" + long + "\nshort\n" + strings.Repeat("y", width+20) + "\n```\n"

		out, err := Markdown(md, width)
		if err != nil {
			t.Fatalf("render at %d: %v", width, err)
		}
		for i, line := range strings.Split(out, "\n") {
			if _, ok := leadingBackground(line); !ok {
				continue
			}
			// A line whose CONTENT genuinely exceeds the pane is the caller's
			// problem — the detail pane hard-wraps it. What must never happen is
			// padding pushing a line over, because that wraps invisible spaces.
			content := ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " "))
			got := ansi.StringWidth(line)
			if content <= width && got != width {
				t.Errorf("width %d: line %d has %d cells of content but is %d wide",
					width, i, content, got)
			}
			if content > width && got != content {
				t.Errorf("width %d: line %d was padded past its own content (%d > %d)",
					width, i, got, content)
			}
		}
	}
}
