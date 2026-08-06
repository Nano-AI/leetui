package render

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// Document is converted content plus the assets referenced from it.
type Document struct {
	// Markdown is ready for Glamour.
	Markdown string

	// Images are the URLs behind each bracket marker, in the order they appear.
	// Marker "[▸ img 2 …]" is Images[1].
	Images []Image
}

// Image is a referenced image.
type Image struct {
	URL string
	Alt string
}

// HTMLToMarkdown converts a LeetCode problem statement or editorial.
func HTMLToMarkdown(input string) (Document, error) {
	node, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return Document{}, fmt.Errorf("parse html: %w", err)
	}
	c := &converter{}
	c.walk(node)
	return Document{
		Markdown: tidy(c.out.String()),
		Images:   c.images,
	}, nil
}

// converter walks the parsed tree and emits markdown.
//
// Element handlers live in inline.go (spans) and blocks.go (paragraphs, lists, code,
// tables, images); whitespace and LaTeX handling live in text.go and latex.go.
type converter struct {
	out    strings.Builder
	images []Image

	// listDepth tracks nesting so nested bullets indent correctly.
	listDepth int
	// ordered tracks whether each list level is numbered, and its counter.
	ordered  []bool
	counters []int
	// inPre suppresses whitespace collapsing inside code blocks.
	inPre bool
}
