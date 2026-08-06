package runner

import "strings"

// The markers, and upgrading a file that predates them. Scaffold.File builds a new file;
// this is the other direction — taking a file that already exists and giving it the same
// shape without touching what is in it.

// Wrap adds scaffolding around a solution file that has none, without touching a byte of
// what is already there.
//
// This is for folders created before scaffolding existed, and for a file whose markers
// were deleted. The existing content becomes the marked region verbatim, so nothing can
// be lost: the only edit is lines added above and below.
//
// The one exception is a Go package clause, which moves OUT of the marked region and into
// the scaffolding. Leaving it inside would submit it, and LeetCode wraps Go submissions in
// a package of its own — see stripPreamble.
//
// Returns changed=false when the file already has markers, so calling this on every edit
// is a no-op after the first.
func (s Scaffold) Wrap(existing string) (string, bool) {
	if HasMarkers(existing) {
		return existing, false
	}
	body := strings.TrimRight(stripPreamble(s.Lang, existing), "\n")
	if strings.TrimSpace(body) == "" {
		return existing, false
	}
	return s.File(body), true
}

// HasMarkers reports whether a file already carries a marked region, in either leetui's
// spelling or vscode-leetcode's.
func HasMarkers(content string) bool {
	return strings.Contains(content, markStart) || strings.Contains(content, vscodeStart)
}
