package render

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdown(t *testing.T) {
	doc, err := HTMLToMarkdown(realStatement)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	md := doc.Markdown
	t.Logf("\n%s", md)

	checks := []struct {
		name string
		want string
	}{
		{"inline code", "`nums`"},
		{"bold", "**exactly one solution**"},
		{"italic", "*same*"},
		{"fenced example", "```"},
		{"example content", "nums = [2,7,11,15], target = 9"},
		{"bullet", "- "},
		{"superscript exponent", "10⁴"},
		{"negative exponent", "-10⁹"},
		{"followup complexity", "O(n²)"},
		{"image bracket", "[▸ img 1 — dp state table]"},
	}
	for _, c := range checks {
		if !strings.Contains(md, c.want) {
			t.Errorf("%s: missing %q", c.name, c.want)
		}
	}

	// Entities must be decoded, not passed through as markup.
	for _, bad := range []string{"&lt;", "&nbsp;", "&gt;", "<p>", "<strong>", "<sup>"} {
		if strings.Contains(md, bad) {
			t.Errorf("raw HTML leaked into markdown: %q", bad)
		}
	}

	if len(doc.Images) != 1 {
		t.Fatalf("collected %d images, want 1", len(doc.Images))
	}
	if got := doc.Images[0].URL; got != "https://assets.leetcode.com/uploads/2021/01/example.png" {
		t.Errorf("image URL = %q", got)
	}
}

// TestImagesAreNumberedInOrder guards the contract behind the bracket markers: pressing
// Enter on "[▸ img 2]" must open Images[1].
func TestImagesAreNumberedInOrder(t *testing.T) {
	in := `<p>one <img src="a.png" alt="first"></p><p>two <img src="b.png" alt="second"></p><p><img src="c.png"></p>`
	doc, err := HTMLToMarkdown(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Images) != 3 {
		t.Fatalf("got %d images, want 3", len(doc.Images))
	}
	for i, want := range []string{"a.png", "b.png", "c.png"} {
		if doc.Images[i].URL != want {
			t.Errorf("Images[%d] = %s, want %s", i, doc.Images[i].URL, want)
		}
	}
	// An image with no alt still gets a numbered marker.
	if !strings.Contains(doc.Markdown, "[▸ img 3]") {
		t.Errorf("unlabeled image lost its marker:\n%s", doc.Markdown)
	}
}
