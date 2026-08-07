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

	reIframe = regexp.MustCompile(`(?is)<iframe[^>]*\bsrc="([^"]+)"[^>]*>.*?</iframe>`)
	reImgTag = regexp.MustCompile(`(?is)<img[^>]*>`)

	// Editorials are markdown with embedded HTML (D-007a), and their figures are
	// MARKDOWN images — `![img](/Figures/134/1.png)` — not <img> tags. Only the tags
	// were handled, so a figure arrived as literal "img /Figures/134/1.png": not a
	// marker anything could open, and not a URL anything could fetch.
	// reAnyMedia matches all three forms in one alternation, so a single pass can number
	// them in the order they appear. Iframes lead, or an <img> nested in one would be
	// matched separately and counted twice.
	reAnyMedia = regexp.MustCompile(`(?is)<iframe[^>]*>.*?</iframe>|<iframe[^>]*/?>|<img[^>]*>|!\[[^\]]*\]\([^)]*\)`)

	reMdImage = regexp.MustCompile(`!\[([^\]]*)\]\(\s*([^)\s]+)(?:\s+"[^"]*")?\s*\)`)
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
	// ONE PASS, so the markers are numbered in DOCUMENT ORDER.
	//
	// Replacing each kind in its own pass numbered every <img> before every markdown
	// image regardless of where they sat, so a figure halfway down could be marker 1
	// and pressing it opened something else entirely. ReplaceAllStringFunc scans
	// left to right, so one alternation over all three keeps the count honest.
	md = reAnyMedia.ReplaceAllStringFunc(md, func(tag string) string {
		url, alt := "", ""

		switch {
		case strings.HasPrefix(strings.ToLower(tag), "<iframe"):
			m := reIframe.FindStringSubmatch(tag)
			if len(m) != 2 {
				return ""
			}
			url, alt = m[1], mediaLabel(m[1])

		case strings.HasPrefix(tag, "!["):
			m := reMdImage.FindStringSubmatch(tag)
			if len(m) != 3 {
				return ""
			}
			alt, url = strings.TrimSpace(m[1]), editorialURL(m[2])
			// LeetCode writes ![img](...) for most figures, which names nothing.
			if alt == "img" {
				alt = ""
			}

		default:
			src := attr(reSrcAttr, tag)
			if src == "" {
				return ""
			}
			url, alt = editorialURL(src), attr(reAltAttr, tag)
		}

		if url == "" {
			return ""
		}
		images = append(images, Image{URL: url, Alt: alt})
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

// editorialFigureBase is where LeetCode serves an editorial's figures.
//
// Their paths arrive RELATIVE — `/Figures/134/1.png` — and resolving them against
// leetcode.com gives a 404. They live under /explore, which is not guessable and was
// found by trying the three candidates against a real figure.
// Spelled out rather than imported: render is a pure formatting package and must not
// depend on the API client, the same reason glamour.go repeats the palette.
const editorialFigureBase = "https://leetcode.com/explore"

// editorialURL makes a figure's path absolute.
//
// A relative path is not something a caller can fetch or a browser can open, so it is
// resolved here rather than at each of the places that would otherwise have to know.
// Anything already absolute is left exactly as it is.
func editorialURL(src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		return src
	case strings.HasPrefix(src, "//"):
		return "https:" + src
	}

	// Editorials write `../Figures/134/1.png`, relative to wherever the card that
	// contains them lives. Joining that to the base verbatim gives
	// `/explore/../Figures/…`, which LeetCode does not normalise and answers 404.
	//
	// Proper URL resolution does not help either: `..` from `/explore/` is
	// `/Figures/…`, and that is a 404 too. The figures are genuinely served from
	// `/explore/Figures/…`, so the traversal is stripped rather than followed.
	// Established by fetching all three candidates, not deduced.
	for {
		switch {
		case strings.HasPrefix(src, "../"):
			src = src[3:]
		case strings.HasPrefix(src, "./"):
			src = src[2:]
		case strings.HasPrefix(src, "/"):
			src = src[1:]
		default:
			return editorialFigureBase + "/" + src
		}
	}
}

// IsDrawable reports whether an Image is an actual picture.
//
// The marker list deliberately mixes figures with code playgrounds and video embeds:
// they are all things a number opens, and renumbering to exclude some would mean the
// key in the pane and the argument on the command line disagreed.
//
// So they stay numbered together, and anything that wants to DRAW one asks first —
// otherwise a playground is fetched as a picture and fails as a bare 403, which
// explains nothing.
func (i Image) IsDrawable() bool {
	switch mediaLabel(i.URL) {
	case "code", "video":
		return false
	}
	return true
}
