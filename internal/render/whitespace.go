package render

import (
	"strings"

	"golang.org/x/net/html"
)

// isBlockContext reports whether a node's children are block-level, i.e. not flowing
// inside a paragraph, list item, table cell, or inline wrapper.
func isBlockContext(parent *html.Node) bool {
	if parent == nil {
		return true
	}
	switch parent.Data {
	case "p", "li", "td", "th", "a", "span", "em", "strong", "b", "i", "code":
		return false
	}
	return true
}

// breakBlock ensures the output ends with a blank line, without stacking more.
func (c *converter) breakBlock() {
	s := c.out.String()
	if s == "" {
		return
	}
	switch {
	case strings.HasSuffix(s, "\n\n"):
	case strings.HasSuffix(s, "\n"):
		c.out.WriteString("\n")
	default:
		c.out.WriteString("\n\n")
	}
}

// space appends a single separator, unless the output already ends with whitespace or
// is empty (no leading space at the start of a block).
func (c *converter) space() {
	s := c.out.String()
	if s == "" {
		return
	}
	switch s[len(s)-1] {
	case ' ', '\n', '\t':
		return
	}
	c.out.WriteString(" ")
}
