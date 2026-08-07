package render

import (
	"strings"
	"testing"
)

// Editorials are markdown with embedded HTML (D-007a), and their figures are MARKDOWN
// images, not <img> tags. Only the tags were handled, so a figure reached the screen as
// literal "img /Figures/134/1.png" — not a marker anything could open, and not a URL
// anything could fetch.
func TestEditorialMarkdownImagesBecomeMarkers(t *testing.T) {
	doc := EditorialToMarkdown("Text.\n\n![img](../Figures/134/1.png)\n")

	if strings.Contains(doc.Markdown, "Figures/134/1.png") {
		t.Errorf("the raw path is still in the text:\n%s", doc.Markdown)
	}
	if !strings.Contains(doc.Markdown, "[▸ 1") {
		t.Errorf("no marker was produced:\n%s", doc.Markdown)
	}
	if len(doc.Images) != 1 {
		t.Fatalf("collected %d images, want 1", len(doc.Images))
	}
	// ../ is stripped rather than followed: /explore/../Figures is a 404, and so is
	// /Figures. The figures genuinely live at /explore/Figures.
	const want = "https://leetcode.com/explore/Figures/134/1.png"
	if doc.Images[0].URL != want {
		t.Errorf("URL = %q, want %q", doc.Images[0].URL, want)
	}
}

// TestMarkersAreNumberedInDocumentOrder is the bug that made pressing a number open the
// wrong picture. Replacing each kind in its own pass numbered every <img> before every
// markdown image regardless of where they sat.
func TestMarkersAreNumberedInDocumentOrder(t *testing.T) {
	doc := EditorialToMarkdown(
		"![img](/Figures/1/a.png)\n\n" +
			`<img src="/Figures/1/b.png" alt="second" />` + "\n\n" +
			"![img](/Figures/1/c.png)\n")

	if len(doc.Images) != 3 {
		t.Fatalf("collected %d images, want 3", len(doc.Images))
	}
	for i, want := range []string{"a.png", "b.png", "c.png"} {
		if !strings.HasSuffix(doc.Images[i].URL, want) {
			t.Errorf("image %d is %s, want %s", i+1, doc.Images[i].URL, want)
		}
	}
	// And the markers must appear in the same order in the text.
	one := strings.Index(doc.Markdown, "[▸ 1")
	two := strings.Index(doc.Markdown, "[▸ 2")
	three := strings.Index(doc.Markdown, "[▸ 3")
	if !(one < two && two < three) {
		t.Errorf("markers are out of order (%d, %d, %d):\n%s", one, two, three, doc.Markdown)
	}
}

func TestEditorialURLShapes(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"../Figures/134/1.png", "https://leetcode.com/explore/Figures/134/1.png"},
		{"/Figures/134/1.png", "https://leetcode.com/explore/Figures/134/1.png"},
		{"./Figures/134/1.png", "https://leetcode.com/explore/Figures/134/1.png"},
		{"Figures/134/1.png", "https://leetcode.com/explore/Figures/134/1.png"},
		// Already absolute: left exactly alone.
		{"https://assets.leetcode.com/x.png", "https://assets.leetcode.com/x.png"},
		{"//assets.leetcode.com/x.png", "https://assets.leetcode.com/x.png"},
		{"", ""},
	} {
		if got := editorialURL(tc.in); got != tc.want {
			t.Errorf("editorialURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A playground or a video is a real marker with a real number — pressing it opens a
// browser. It is just not a picture, and drawing one fails as a bare 403.
func TestPlaygroundsAreNotDrawable(t *testing.T) {
	for _, tc := range []struct {
		url      string
		drawable bool
	}{
		{"https://leetcode.com/playground/QVD5XPPc/shared", false},
		{"https://player.vimeo.com/video/123", false},
		{"https://www.youtube.com/embed/abc", false},
		{"https://leetcode.com/explore/Figures/134/1.png", true},
	} {
		if got := (Image{URL: tc.url}).IsDrawable(); got != tc.drawable {
			t.Errorf("%s: drawable = %v, want %v", tc.url, got, tc.drawable)
		}
	}
}
