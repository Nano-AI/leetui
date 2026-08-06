package runner

import "strings"

// ExtractCode returns the part of a solution file that should be submitted.
//
// This is the other half of the scaffolding bargain (see scaffold.go). The file on disk
// carries imports, a package clause, and a driver include so the editor and the local
// compiler are happy. LeetCode supplies all of that itself, and some of it is an outright
// compile error there — a Go submission carrying `package main` is rejected, because the
// judge wraps the code in a package of its own.
//
// Three cases, in order:
//
//	markers present   return exactly the marked region
//	no markers        fall back to the whole file, minus anything the judge cannot
//	                  accept — this is what a file written before the markers existed,
//	                  or one whose markers were deleted, looks like
//	nothing marked    the whole file, unchanged
func ExtractCode(lang Lang, content string) string {
	if body, ok := betweenMarkers(content); ok {
		return body
	}
	return stripPreamble(lang, content)
}

// LegacyStarter is the solution file leetui wrote BEFORE scaffolding existed: the raw
// snippet, with a bare package clause for Go and nothing for anything else.
//
// It exists for one comparison — telling an untouched starter file apart from one someone
// has typed in — so an old folder can be upgraded to a scaffolded file without ever
// risking work. Do not use it to write anything; that is Scaffold's job.
func LegacyStarter(lang Lang, snippet string) string {
	if lang.Slug == "golang" {
		return "package main\n\n" + snippet
	}
	return snippet
}

// betweenMarkers pulls out the region between a start and end marker.
//
// Both leetui's markers and vscode-leetcode's are recognised, so a workspace built with
// that extension submits correctly without being rewritten. A start with no end takes
// everything after it: a truncated marker pair is far more likely to be a stray edit than
// a request to submit nothing.
func betweenMarkers(content string) (string, bool) {
	lines := strings.Split(content, "\n")

	start, end := -1, -1
	for i, l := range lines {
		switch {
		case start < 0 && (strings.Contains(l, markStart) || strings.Contains(l, vscodeStart)):
			start = i
		case start >= 0 && end < 0 && (strings.Contains(l, markEnd) || strings.Contains(l, vscodeEnd)):
			end = i
		}
	}
	if start < 0 {
		return "", false
	}
	if end < 0 {
		end = len(lines)
	}
	return strings.Trim(strings.Join(lines[start+1:end], "\n"), "\n") + "\n", true
}

// stripPreamble removes scaffolding the judge would reject from an unmarked file.
//
// Only Go needs this, and only for the package clause. Everything else — C++ includes,
// Python imports — is accepted by LeetCode and harmless to send.
func stripPreamble(lang Lang, content string) string {
	if lang.Slug != "golang" {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		if strings.HasPrefix(t, "package ") {
			return strings.TrimLeft(strings.Join(lines[i+1:], "\n"), "\n")
		}
		// The first real line is not a package clause, so there is nothing to strip.
		return content
	}
	return content
}
