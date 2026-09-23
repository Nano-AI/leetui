package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMarkJSONShapeIsStable pins the field names. Agents parse this output, so renaming a
// key is a breaking change and has to be made in docs/AGENTS.md first.
func TestMarkJSONShapeIsStable(t *testing.T) {
	b, err := json.Marshal(markItem{
		Slug: "two-sum", Mark: "up", Title: "Two Sum", Difficulty: "Easy",
		ID: 1, Status: "ac", Note: "the hash-map insight",
		MarkedAt: "2026-08-09T12:00:00Z",
		URL:      "https://leetcode.com/problems/two-sum/",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{
		`"slug":"two-sum"`, `"mark":"up"`, `"title":"Two Sum"`,
		`"difficulty":"Easy"`, `"id":1`, `"status":"ac"`,
		`"note":"the hash-map insight"`, `"marked_at":"2026-08-09T12:00:00Z"`,
		`"url":"https://leetcode.com/problems/two-sum/"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s in %s", want, b)
		}
	}
}

// A mark for a problem this machine has not synced still carries the two fields that
// matter, and omits the rest rather than emitting empty strings.
func TestMarkJSONOmitsWhatItDoesNotKnow(t *testing.T) {
	b, err := json.Marshal(markItem{Slug: "not-synced", Mark: "down", MarkedAt: "2026-08-09T12:00:00Z"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"slug":"not-synced"`) || !strings.Contains(got, `"mark":"down"`) {
		t.Errorf("the essentials are missing: %s", got)
	}
	for _, absent := range []string{`"title"`, `"difficulty"`, `"id"`, `"status"`, `"url"`} {
		if strings.Contains(got, absent) {
			t.Errorf("%s should have been omitted: %s", absent, got)
		}
	}
}

// TestMarkLineLeadsWithTheDirection: a list of thirty is scanned down its left edge, the
// way the board's MARK column is.
func TestMarkLineLeadsWithTheDirection(t *testing.T) {
	up := markLine(markItem{Slug: "two-sum", Mark: "up", Title: "Two Sum", ID: 1,
		Difficulty: "Easy", Status: "ac", Note: "redo"})
	if !strings.HasPrefix(up, "+") {
		t.Errorf("an important line starts %q, want a leading +", up[:1])
	}
	for _, want := range []string{"Two Sum", "easy", "solved", "redo"} {
		if !strings.Contains(up, want) {
			t.Errorf("the line is missing %q: %s", want, up)
		}
	}

	down := markLine(markItem{Slug: "lru-cache", Mark: "down", Title: "LRU Cache", ID: 146})
	if !strings.HasPrefix(down, "-") {
		t.Errorf("an unimportant line starts %q, want a leading -", down[:1])
	}

	// A mark on something unsynced says so rather than showing a bare slug that reads
	// like a title.
	unknown := markLine(markItem{Slug: "mystery", Mark: "up"})
	if !strings.Contains(unknown, "not synced") {
		t.Errorf("an unsynced entry does not say so: %s", unknown)
	}
}

// TestBareDashIsADirection: `leetui mark - two-sum` must reach markSet rather than being
// routed to the list as a malformed flag.
func TestBareDashIsADirection(t *testing.T) {
	if !isDirection("-") {
		t.Error(`"-" must read as the down direction`)
	}
	for _, flagish := range []string{"--json", "-json", "--up"} {
		if isDirection(flagish) {
			t.Errorf("%q is a flag, not a direction", flagish)
		}
	}
}
