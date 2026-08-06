package render

import (
	"fmt"
	"regexp"
	"strings"
)

// Editorial rendering (D-007).
//
// An editorial is NOT a problem statement. Statements are HTML; editorials are markdown
// with HTML embedded in it — figures, Vimeo players, and LeetCode "playground" iframes
// holding the reference implementation. Feeding one to HTMLToMarkdown would parse the
// prose as a text node and lose every heading.
//
// So the pipeline is the other way round: keep the markdown, and reduce the embedded
// HTML to the same bracket markers the statement renderer uses. A marker is openable —
// pressing its number hands the URL to the browser — which is the honest floor for a
// video or a code playground that a terminal cannot show inline.

var (
	// LeetCode puts a [TOC] marker at the top of most editorials. The site turns it into
	// a sidebar; Glamour renders it literally as "[TOC]".
	reTOC = regexp.MustCompile(`(?m)^\s*\[TOC\]\s*$`)

	reIframe  = regexp.MustCompile(`(?is)<iframe[^>]*\bsrc="([^"]+)"[^>]*>.*?</iframe>`)
	reImgTag  = regexp.MustCompile(`(?is)<img[^>]*>`)
	reSrcAttr = regexp.MustCompile(`(?i)\bsrc="([^"]+)"`)
	reAltAttr = regexp.MustCompile(`(?i)\balt="([^"]*)"`)

	// Layout-only wrappers. LeetCode nests figures in <div class="video-container"> and
	// friends; once the media inside is a marker, the wrapper carries nothing.
	reBareDiv = regexp.MustCompile(`(?i)</?div[^>]*>`)
	reCenter  = regexp.MustCompile(`(?i)</?(center|figure|figcaption)[^>]*>`)

	// Three or more blank lines, left behind once wrappers are stripped.
	reBlankRun = regexp.MustCompile(`\n{3,}`)
)

// Editorial converts an official solution's markdown and renders it at the given width,
// returning the assets behind each bracket marker in the order they appear.
func Editorial(md string, width int) (string, []Image, error) {
	doc := EditorialToMarkdown(md)
	out, err := Markdown(doc.Markdown, width)
	if err != nil {
		return "", doc.Images, err
	}
	return out, doc.Images, nil
}

// EditorialToMarkdown reduces an editorial's embedded HTML to markers.
//
// Split out from Editorial so it can be tested without a terminal renderer, and so a
// caller that wants the markdown (a future "save the editorial next to the solution")
// does not have to render it first.
func EditorialToMarkdown(md string) Document {
	var images []Image

	// Order matters: iframes are matched before <img>, because a stripped <div> would
	// otherwise leave an iframe's attributes loose in the text.
	md = reIframe.ReplaceAllStringFunc(md, func(tag string) string {
		m := reIframe.FindStringSubmatch(tag)
		if len(m) != 2 {
			return ""
		}
		url := m[1]
		images = append(images, Image{URL: url, Alt: mediaLabel(url)})
		return marker(len(images), mediaLabel(url))
	})

	md = reImgTag.ReplaceAllStringFunc(md, func(tag string) string {
		src := attr(reSrcAttr, tag)
		if src == "" {
			return ""
		}
		alt := attr(reAltAttr, tag)
		images = append(images, Image{URL: src, Alt: alt})
		if alt == "" {
			return marker(len(images), "figure")
		}
		return marker(len(images), alt)
	})

	md = reTOC.ReplaceAllString(md, "")
	md = reBareDiv.ReplaceAllString(md, "")
	md = reCenter.ReplaceAllString(md, "")
	md = strings.ReplaceAll(md, "&nbsp;", " ")

	md = approximateLaTeX(md)
	md = reBlankRun.ReplaceAllString(md, "\n\n")

	return Document{Markdown: strings.TrimSpace(md) + "\n", Images: images}
}

// marker renders an openable bracket, matching the statement renderer's format so both
// panes teach the same gesture: the number opens it.
//
// Wrapped in backticks for the same reason as in blocks.go — inline code is the one
// Glamour style that survives being adjacent to prose without fusing into it.
func marker(n int, label string) string {
	if label == "" {
		return fmt.Sprintf("\n\n`[▸ %d]`\n\n", n)
	}
	return fmt.Sprintf("\n\n`[▸ %d — %s]`\n\n", n, label)
}

// mediaLabel names an embed by what it is, read off the URL. The user is deciding
// whether to leave the terminal for it, and "video" versus "code" is the whole decision.
func mediaLabel(url string) string {
	switch {
	case strings.Contains(url, "/playground/"):
		return "code"
	case strings.Contains(url, "vimeo") || strings.Contains(url, "youtube"):
		return "video"
	default:
		return "embed"
	}
}

func attr(re *regexp.Regexp, tag string) string {
	if m := re.FindStringSubmatch(tag); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
