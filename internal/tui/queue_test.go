package tui

import (
	"testing"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// The verdict answers "did it pass". These answer the question asked half a second
// later, which previously required a trip to leetcode.com.
func TestQueueStats(t *testing.T) {
	for _, tc := range []struct {
		name string
		item queueItem
		want string
	}{
		{
			"accepted with figures",
			queueItem{Verdict: theme.Accepted, Runtime: "58 ms", Memory: "20.2 MB", Percentile: 91.23},
			"58 ms · beats 91% · 20.2 MB",
		},
		{
			// A real 0.0% and a missing one are the same value in the payload, so
			// claiming to beat nobody is worse than staying quiet.
			"accepted, no percentile",
			queueItem{Verdict: theme.Accepted, Runtime: "3 ms", Memory: "8.1 MB"},
			"3 ms · 8.1 MB",
		},
		{
			// The judge sometimes reports neither figure. The row must not render a
			// stray separator with nothing on either side of it.
			"accepted, nothing reported",
			queueItem{Verdict: theme.Accepted},
			"",
		},
		{
			// "wrong on 3 of 63" and "wrong on 62 of 63" call for different next moves,
			// and nothing else on screen tells them apart.
			"wrong answer shows how far it got",
			queueItem{Verdict: theme.WrongAnswer, Correct: 62, Total: 63},
			"62/63 cases",
		},
		{
			"still pending",
			queueItem{Verdict: theme.Pending},
			"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.item.stats(); got != tc.want {
				t.Errorf("stats() = %q, want %q", got, tc.want)
			}
		})
	}
}
