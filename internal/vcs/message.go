package vcs

import (
	"fmt"
	"strings"
)

// The commit message convention (D-011):
//
//	solve(0146): lru cache — go, 58ms, beats 91.2%
//
// Short, specific, developer syntax. The scope is the zero-padded problem number so a
// `git log --oneline` sorts and greps the same way the workspace folders do.
//
// **No trailers.** These commits record the user solving a problem; leetui typed none of
// it and neither did any model, so nothing is co-authored and there is no generated-with
// footer or session link (D-024).

// Message describes a solved problem, for building the commit subject.
type Message struct {
	ID    int
	Title string

	// Lang is the display name, e.g. "Go". Lowercased in the subject.
	Lang string

	// Runtime is the judge's figure, e.g. "58 ms". Optional.
	Runtime string

	// Percentile is the runtime beat percentage. Zero omits it — a real 0.0% and an
	// absent figure are indistinguishable in the payload, and claiming to beat nobody
	// is worse than staying quiet.
	Percentile float64

	// Note is an optional body paragraph.
	Note string
}

// Subject renders the one-line summary.
func (m Message) Subject() string {
	var b strings.Builder
	fmt.Fprintf(&b, "solve(%04d): %s", m.ID, strings.ToLower(strings.TrimSpace(m.Title)))

	var facts []string
	if lang := strings.ToLower(strings.TrimSpace(m.Lang)); lang != "" {
		facts = append(facts, lang)
	}
	if rt := compactRuntime(m.Runtime); rt != "" {
		facts = append(facts, rt)
	}
	if m.Percentile > 0 {
		facts = append(facts, fmt.Sprintf("beats %.1f%%", m.Percentile))
	}
	if len(facts) > 0 {
		fmt.Fprintf(&b, " — %s", strings.Join(facts, ", "))
	}
	return b.String()
}

// String renders the full message: subject, blank line, note.
func (m Message) String() string {
	subject := m.Subject()
	note := strings.TrimSpace(m.Note)
	if note == "" {
		return subject
	}
	return subject + "\n\n" + note + "\n"
}

// compactRuntime turns the judge's "58 ms" into "58ms".
//
// The space is how LeetCode formats it for a web page with room to spare. A subject line
// has 50 usable columns and the units are unambiguous without it.
func compactRuntime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return ""
	}
	return strings.ReplaceAll(s, " ", "")
}
