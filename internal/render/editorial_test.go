package render

import (
	"strings"
	"testing"
)

// realEditorial is the shape LeetCode actually serves for question.solution.content,
// taken from Two Sum on 2026-08-06: markdown prose with a [TOC] marker, HTML-wrapped
// media, and $$…$$ display math.
const realEditorial = `[TOC]

## Video Solution

---

<div>
    <div class="video-container">
        <iframe src="https://player.vimeo.com/video/567281997" width="640" height="360" frameborder="0" allowfullscreen></iframe>
    </div>
</div>

<div>&nbsp;
</div>

## Solution Article

---

### Approach 1: Brute Force

**Algorithm**

Loop through each element $$x$$ and find if there is another value that equals $$target - x$$.

**Implementation**

<iframe src="https://leetcode.com/playground/WTVGRyeD/shared" frameBorder="0" width="100%" height="327" name="WTVGRyeD"></iframe>

**Complexity Analysis**

* Time complexity: $$O(n^2)$$.
* Space complexity: $$O(1)$$.

<img alt="hash table walkthrough" src="https://assets.leetcode.com/uploads/2021/01/two-sum.png" />
`

func TestEditorialMarkersAreOpenable(t *testing.T) {
	doc := EditorialToMarkdown(realEditorial)

	if len(doc.Images) != 3 {
		t.Fatalf("got %d openable assets, want 3 (video, playground, figure): %+v",
			len(doc.Images), doc.Images)
	}

	// The order of the slice IS the marker numbering — pressing 2 must open the second
	// marker, not the second image.
	want := []string{
		"https://player.vimeo.com/video/567281997",
		"https://leetcode.com/playground/WTVGRyeD/shared",
		"https://assets.leetcode.com/uploads/2021/01/two-sum.png",
	}
	for i, w := range want {
		if doc.Images[i].URL != w {
			t.Errorf("asset %d = %q, want %q", i+1, doc.Images[i].URL, w)
		}
	}

	for _, marker := range []string{"[▸ 1 — video]", "[▸ 2 — code]", "[▸ 3 — hash table walkthrough]"} {
		if !strings.Contains(doc.Markdown, marker) {
			t.Errorf("missing marker %q in:\n%s", marker, doc.Markdown)
		}
	}
}

func TestEditorialStripsSiteChrome(t *testing.T) {
	doc := EditorialToMarkdown(realEditorial)

	for _, junk := range []string{"[TOC]", "<div", "</div>", "<iframe", "<img", "&nbsp;"} {
		if strings.Contains(doc.Markdown, junk) {
			t.Errorf("%q survived conversion:\n%s", junk, doc.Markdown)
		}
	}

	// Headings are the reason this file exists: running an editorial through the HTML
	// converter would flatten them into one text node.
	for _, heading := range []string{"## Video Solution", "### Approach 1: Brute Force"} {
		if !strings.Contains(doc.Markdown, heading) {
			t.Errorf("lost heading %q", heading)
		}
	}
}

func TestEditorialDisplayMath(t *testing.T) {
	doc := EditorialToMarkdown(realEditorial)

	// $$O(n^2)$$ must not leave stray dollars behind. That is exactly what the inline
	// rule does on its own, which is why reDisplayMath runs first.
	if strings.Contains(doc.Markdown, "$") {
		t.Errorf("a dollar sign survived display math:\n%s", doc.Markdown)
	}
	if !strings.Contains(doc.Markdown, "O(n²)") {
		t.Errorf("want O(n²) from $$O(n^2)$$, got:\n%s", doc.Markdown)
	}
}

func TestEditorialRendersAtWidth(t *testing.T) {
	out, images, err := Editorial(realEditorial, 60)
	if err != nil {
		t.Fatalf("Editorial: %v", err)
	}
	if len(images) != 3 {
		t.Errorf("got %d images, want 3", len(images))
	}
	for i, line := range strings.Split(out, "\n") {
		if w := visibleWidth(line); w > 60 {
			t.Errorf("line %d is %d cells wide, over the 60 requested: %q", i+1, w, line)
		}
	}
}
