package render

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func (c *converter) list(n *html.Node, ordered bool) {
	c.listDepth++
	c.ordered = append(c.ordered, ordered)
	c.counters = append(c.counters, 0)

	c.children(n)

	c.listDepth--
	c.ordered = c.ordered[:len(c.ordered)-1]
	c.counters = c.counters[:len(c.counters)-1]
	c.out.WriteString("\n")
}

func (c *converter) listItem(n *html.Node) {
	depth := c.listDepth - 1
	if depth < 0 {
		depth = 0
	}
	indent := strings.Repeat("  ", depth)

	marker := "- "
	if len(c.ordered) > 0 && c.ordered[len(c.ordered)-1] {
		c.counters[len(c.counters)-1]++
		marker = fmt.Sprintf("%d. ", c.counters[len(c.counters)-1])
	}

	var inner converter
	inner.listDepth = c.listDepth
	inner.ordered = c.ordered
	inner.counters = c.counters
	inner.children(n)
	c.images = append(c.images, inner.images...)

	body := strings.TrimSpace(inner.out.String())
	if body == "" {
		return
	}
	c.out.WriteString(indent + marker + body + "\n")
}

// image emits a bracket marker and records the URL.
//
// The bracket is the floor: it works in every terminal, degrades to a readable label,
// and gives the user something to press Enter on. Inline graphics layer on top for
// terminals that support them; they never replace this.
func (c *converter) image(n *html.Node) {
	var src, alt string
	for _, a := range n.Attr {
		switch a.Key {
		case "src":
			src = a.Val
		case "alt":
			alt = a.Val
		}
	}
	if src == "" {
		return
	}
	c.images = append(c.images, Image{URL: src, Alt: alt})

	// An <img> sitting directly in the body (not inside a paragraph or list item) is a
	// block-level diagram. Without explicit separation its marker fuses onto whatever
	// text follows: "[▸ img 1 — dp table] Example 1:".
	if isBlockContext(n.Parent) {
		c.breakBlock()
	}
	defer func() {
		if isBlockContext(n.Parent) {
			c.out.WriteString("\n\n")
		}
	}()

	label := strings.TrimSpace(alt)
	n_ := len(c.images)
	if label == "" {
		c.out.WriteString(fmt.Sprintf("`[▸ img %d]`", n_))
	} else {
		c.out.WriteString(fmt.Sprintf("`[▸ img %d — %s]`", n_, label))
	}
}

func (c *converter) link(n *html.Node) {
	var href string
	for _, a := range n.Attr {
		if a.Key == "href" {
			href = a.Val
		}
	}
	var inner converter
	inner.children(n)
	text := strings.TrimSpace(inner.out.String())
	c.images = append(c.images, inner.images...)

	switch {
	case text == "":
		return
	case href == "":
		c.out.WriteString(text)
	default:
		c.out.WriteString("[" + text + "](" + href + ")")
	}
}

func (c *converter) superscript(n *html.Node) {
	var inner converter
	inner.children(n)
	c.out.WriteString(toSuperscript(strings.TrimSpace(inner.out.String())))
}

func (c *converter) subscript(n *html.Node) {
	var inner converter
	inner.children(n)
	c.out.WriteString(toSubscript(strings.TrimSpace(inner.out.String())))
}

// table emits a markdown table. LeetCode uses these rarely (mostly in editorials), and
// Glamour renders them well, so a straightforward row/cell walk is enough.
func (c *converter) table(n *html.Node) {
	rows := collectRows(n)
	if len(rows) == 0 {
		return
	}
	c.out.WriteString("\n")
	for i, row := range rows {
		c.out.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			c.out.WriteString("|" + strings.Repeat(" --- |", len(row)) + "\n")
		}
	}
	c.out.WriteString("\n")
}

func collectRows(n *html.Node) [][]string {
	var rows [][]string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			var cells []string
			for cell := node.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Type != html.ElementNode || (cell.Data != "td" && cell.Data != "th") {
					continue
				}
				var inner converter
				inner.children(cell)
				cells = append(cells, strings.TrimSpace(collapse(inner.out.String())))
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
			return
		}
		for ch := node.FirstChild; ch != nil; ch = ch.NextSibling {
			visit(ch)
		}
	}
	visit(n)
	return rows
}
