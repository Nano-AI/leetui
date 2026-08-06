package render

import (
	"strings"
	"testing"
)

// --- Regressions -------------------------------------------------------------
//
// These three all rendered wrong at first and passed the original assertions, which
// only checked that expected substrings existed. They check what must NOT appear.

// TestNoLiteralMarkdownInsideFences: LeetCode wraps the Input/Output/Explanation labels
// in <strong> inside every <pre>. Markdown is not parsed inside a fence, so emitting
// "**Input:**" printed literal asterisks in every example block.
func TestNoLiteralMarkdownInsideFences(t *testing.T) {
	doc, err := HTMLToMarkdown(realStatement)
	if err != nil {
		t.Fatal(err)
	}
	inFence := false
	for _, line := range strings.Split(doc.Markdown, "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			continue
		}
		for _, marker := range []string{"**", "`", "*"} {
			if strings.Contains(line, marker) {
				t.Errorf("literal %q inside code fence: %q", marker, line)
			}
		}
	}
	if !strings.Contains(doc.Markdown, "Input: nums = [2,7,11,15]") {
		t.Error("example label lost its text")
	}
}

// TestInlineElementsKeepSpacing: inline elements are rendered by a sub-converter that
// trims, so the surrounding text node's edge whitespace is the only record of the gap.
// Dropping it produced "sameelement" and "-10⁹<= nums[i]".
func TestInlineElementsKeepSpacing(t *testing.T) {
	doc, err := HTMLToMarkdown(realStatement)
	if err != nil {
		t.Fatal(err)
	}
	for _, fused := range []string{"sameelement", "⁹<=", "⁴<=", "Follow-up:**Can"} {
		if strings.Contains(doc.Markdown, fused) {
			t.Errorf("text fused across an inline element: %q", fused)
		}
	}
	if !strings.Contains(doc.Markdown, "*same* element") {
		t.Errorf("expected a space after the emphasized word:\n%s", doc.Markdown)
	}
}

// TestBlockImageStandsAlone: an <img> directly in the body is a diagram, and its marker
// was fusing onto the next paragraph ("[▸ img 1 — dp state table] Example 1:").
func TestBlockImageStandsAlone(t *testing.T) {
	doc, err := HTMLToMarkdown(realStatement)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(doc.Markdown, "\n") {
		if !strings.Contains(line, "img 1") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "`[▸ img 1 — dp state table]`" {
			t.Errorf("block image shares its line with other content: %q", trimmed)
		}
		return
	}
	t.Error("image marker not found at all")
}

// TestInlineImageStaysInline is the counterpart: an image inside a paragraph must NOT
// be broken onto its own line, or prose gets chopped in half.
func TestInlineImageStaysInline(t *testing.T) {
	doc, err := HTMLToMarkdown(`<p>before <img src="x.png" alt="x"> after</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "before `[▸ img 1 — x]` after") {
		t.Errorf("inline image broke its paragraph:\n%q", doc.Markdown)
	}
}

// TestNonBreakingSpaceIsNormalized: LeetCode emits &nbsp; for layout. Left as U+00A0 it
// renders as a stray glyph and defeats word wrapping.
func TestNonBreakingSpaceIsNormalized(t *testing.T) {
	doc, err := HTMLToMarkdown("<p>a b  c</p>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(doc.Markdown, " ") {
		t.Error("non-breaking space survived into the output")
	}
	if !strings.Contains(doc.Markdown, "a b c") {
		t.Errorf("nbsp did not become a plain space: %q", doc.Markdown)
	}
}
