package vcs

import (
	"strings"
	"testing"
)

func TestSubjectFollowsTheConvention(t *testing.T) {
	got := Message{
		ID: 146, Title: "LRU Cache", Lang: "Go",
		Runtime: "58 ms", Percentile: 91.23,
	}.Subject()

	// The convention recorded in D-011, character for character.
	want := "solve(0146): lru cache — go, 58ms, beats 91.2%"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestSubjectOmitsFiguresItDoesNotHave(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  Message
		want string
	}{
		{
			// A local commit, never submitted: no judge, so no figures.
			"no figures",
			Message{ID: 1, Title: "Two Sum"},
			"solve(0001): two sum",
		},
		{
			"language only",
			Message{ID: 1, Title: "Two Sum", Lang: "Python3"},
			"solve(0001): two sum — python3",
		},
		{
			// A real 0.0% and an absent percentile are the same value in the
			// payload. Claiming to beat nobody is worse than staying quiet.
			"zero percentile is omitted",
			Message{ID: 1, Title: "Two Sum", Lang: "Go", Percentile: 0},
			"solve(0001): two sum — go",
		},
		{
			"the judge's N/A runtime",
			Message{ID: 1, Title: "Two Sum", Lang: "Go", Runtime: "N/A"},
			"solve(0001): two sum — go",
		},
		{
			// Contest-series problems have no number. Padding still applies, and
			// they sort to the front exactly as their folders do.
			"unnumbered problem",
			Message{ID: 0, Title: "LCP 01"},
			"solve(0000): lcp 01",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.msg.Subject(); got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestMessageBodyIsSeparatedByABlankLine(t *testing.T) {
	got := Message{ID: 1, Title: "Two Sum", Note: "hash map, one pass"}.String()
	want := "solve(0001): two sum\n\nhash map, one pass\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestNoTrailers pins D-024. These commits record the user solving a problem; leetui
// typed none of it and neither did any model. A co-author trailer would be a false
// attribution on someone else's repository, and a session link would be worse.
func TestNoTrailers(t *testing.T) {
	full := Message{
		ID: 146, Title: "LRU Cache", Lang: "Go",
		Runtime: "58 ms", Percentile: 91.2, Note: "doubly linked list",
	}.String()

	for _, banned := range []string{
		"Co-Authored-By", "Generated with", "claude.ai", "Claude", "🤖",
	} {
		if strings.Contains(full, banned) {
			t.Errorf("commit message contains %q:\n%s", banned, full)
		}
	}
}
