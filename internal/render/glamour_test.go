package render

import (
	"strings"
	"testing"
)

func TestGlamourRenders(t *testing.T) {
	out, images, err := HTML(realStatement, 70)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(images) != 1 {
		t.Errorf("got %d images, want 1", len(images))
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("rendered output is empty")
	}
	t.Logf("\n%s", out)

	// Every line must fit the requested width, or the detail pane overflows.
	for i, line := range strings.Split(out, "\n") {
		if w := visibleWidth(line); w > 74 { // wrap width + Glamour's own margin
			t.Errorf("line %d is %d cols wide, exceeds wrap width", i, w)
		}
	}
}

func TestEmptyAndMalformedInput(t *testing.T) {
	for _, in := range []string{"", "   ", "<p>", "<<<>>>", "plain text, no tags"} {
		if _, err := HTMLToMarkdown(in); err != nil {
			t.Errorf("HTMLToMarkdown(%q) errored: %v", in, err)
		}
	}
}

// visibleWidth counts display cells, skipping ANSI escape sequences.
func visibleWidth(s string) int {
	n, inEsc := 0, false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || r == 'K'):
			inEsc = false
		case !inEsc:
			n++
		}
	}
	return n
}
