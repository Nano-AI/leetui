package render

import (
	"strings"

	"golang.org/x/net/html"
)

func (c *converter) inline(n *html.Node, open, close string) {
	var inner converter
	inner.inPre = c.inPre
	inner.children(n)
	body := strings.TrimSpace(inner.out.String())
	if body == "" {
		return
	}
	c.images = append(c.images, inner.images...)
	c.out.WriteString(open + body + close)
}

func (c *converter) block(n *html.Node, open, close string) {
	start := c.out.Len()
	c.out.WriteString(open)
	c.children(n)
	if c.out.Len() == start+len(open) {
		return // empty block; drop the marker rather than emit a bare "###"
	}
	c.out.WriteString(close + "\n\n")
}

func (c *converter) pre(n *html.Node) {
	c.inPre = true
	var inner converter
	inner.inPre = true
	inner.children(n)
	c.inPre = false

	body := strings.Trim(inner.out.String(), "\n")
	if body == "" {
		return
	}
	// Examples are the most-read part of a statement, so they get a fenced block with
	// no language: Chroma highlighting a stray "Input: [2,7]" as code reads worse than
	// leaving it plain.
	c.out.WriteString("\n```\n" + body + "\n```\n\n")
}
