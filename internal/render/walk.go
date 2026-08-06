package render

import "golang.org/x/net/html"

func (c *converter) walk(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		c.text(n.Data)
		return

	case html.ElementNode:
		// Inside a fenced code block markdown is not parsed, so emitting "**bold**"
		// would print literal asterisks. LeetCode wraps the Input/Output/Explanation
		// labels in <strong> inside every <pre>, so this is the common case, not an
		// edge case. Emit the children and drop the formatting.
		if c.inPre {
			switch n.Data {
			case "strong", "b", "em", "i", "del", "s", "code", "span", "font":
				c.children(n)
				return
			}
		}

		switch n.Data {
		case "script", "style", "head":
			return // never render

		case "br":
			c.out.WriteString("\n")
			return

		case "img":
			c.image(n)
			return

		case "hr":
			c.out.WriteString("\n\n---\n\n")
			return

		case "p", "div":
			c.block(n, "", "")
			return

		case "h1":
			c.block(n, "\n## ", "\n")
			return
		case "h2", "h3":
			c.block(n, "\n### ", "\n")
			return
		case "h4", "h5", "h6":
			c.block(n, "\n#### ", "\n")
			return

		case "strong", "b":
			c.inline(n, "**", "**")
			return
		case "em", "i":
			c.inline(n, "*", "*")
			return
		case "del", "s":
			c.inline(n, "~~", "~~")
			return

		case "code":
			// A <code> inside <pre> is the block's content, not an inline span.
			if c.inPre {
				c.children(n)
				return
			}
			c.inline(n, "`", "`")
			return

		case "pre":
			c.pre(n)
			return

		case "ul", "ol":
			c.list(n, n.Data == "ol")
			return

		case "li":
			c.listItem(n)
			return

		case "sup":
			c.superscript(n)
			return
		case "sub":
			c.subscript(n)
			return

		case "a":
			c.link(n)
			return

		case "table":
			c.table(n)
			return
		}
	}

	c.children(n)
}

func (c *converter) children(n *html.Node) {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		c.walk(ch)
	}
}

func (c *converter) text(s string) {
	if c.inPre {
		c.out.WriteString(s)
		return
	}

	// Whitespace outside code is collapsed — LeetCode's HTML is pretty-printed, and
	// its newlines are markup, not content.
	//
	// But leading and trailing whitespace must survive as a single space. Inline
	// elements are rendered by a sub-converter that trims its result, so the only
	// record that "<em>same</em> element" had a gap is the leading space on the text
	// node that follows. Dropping it produces "sameelement".
	lead := startsWithSpace(s)
	trail := endsWithSpace(s)
	core := collapse(s)

	if core == "" {
		if lead || trail {
			c.space()
		}
		return
	}
	if lead {
		c.space()
	}
	c.out.WriteString(approximateLaTeX(core))
	if trail {
		c.out.WriteString(" ")
	}
}
