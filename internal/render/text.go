package render

import "strings"

// ---------------------------------------------------------------------------
// Text helpers
// ---------------------------------------------------------------------------

// isSpace covers ASCII whitespace plus U+00A0, which LeetCode emits as &nbsp; for
// layout. It must not survive into the output as a literal non-breaking space, or it
// renders as a stray glyph and breaks word wrapping.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\u00a0'
}

// collapse squeezes internal whitespace runs to a single space and strips the edges.
// Edge whitespace is preserved by the caller via startsWithSpace/endsWithSpace.
func collapse(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if isSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteRune(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

func startsWithSpace(s string) bool {
	for _, r := range s {
		return isSpace(r)
	}
	return false
}

func endsWithSpace(s string) bool {
	r := []rune(s)
	if len(r) == 0 {
		return false
	}
	return isSpace(r[len(r)-1])
}

// tidy normalizes blank runs so Glamour's spacing stays even.
func tidy(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, l := range lines {
		l = strings.TrimRight(l, " \t")
		if l == "" {
			blanks++
			if blanks > 1 {
				continue
			}
		} else {
			blanks = 0
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}
