package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/store"
)

// Reading the marks out, for a person and for a program.
//
// The JSON shape is documented as stable in docs/AGENTS.md — agents parse it, so renaming
// a field is a breaking change and belongs in that file first.

func markList(a *app, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui mark list", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print the marks as JSON, for scripts and agents")
	onlyUp := fs.Bool("up", false, "only the important ones")
	onlyDown := fs.Bool("down", false, "only the unimportant ones")
	if _, err := parseFlags(fs, args); err != nil {
		return exitProblem, err
	}
	if *onlyUp && *onlyDown {
		return exitProblem, fmt.Errorf("--up and --down are opposites; pass neither for both")
	}

	want := store.MarkNone
	switch {
	case *onlyUp:
		want = store.MarkUp
	case *onlyDown:
		want = store.MarkDown
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := a.store.Marks(ctx, want)
	if err != nil {
		return exitProblem, err
	}

	items := make([]markItem, 0, len(entries))
	for _, e := range entries {
		it := markItem{
			Slug:     e.Slug,
			Mark:     string(e.Mark),
			Note:     e.Note,
			MarkedAt: e.MarkedAt.UTC().Format(time.RFC3339),
		}
		// A mark may name a problem this machine has not synced; the verdict still holds,
		// it just has less to say about it.
		if d, err := a.store.Get(ctx, e.Slug); err == nil {
			it.Title, it.Difficulty, it.ID, it.Status = d.Title, d.Difficulty, d.NumericID, d.Status
			it.URL = problemURL(d)
		}
		items = append(items, it)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		// Always an array, never null: a script piping this into a loop should not have
		// to special-case an empty result.
		return exitOK, enc.Encode(items)
	}

	if len(items) == 0 {
		fmt.Println("Nothing marked. Mark one: leetui mark up two-sum")
		return exitOK, nil
	}
	for _, it := range items {
		fmt.Println(markLine(it))
	}
	return exitOK, nil
}

// markLine is one verdict as a human reads it.
//
// The direction leads, as a glyph in the first column, so a list of thirty scans down the
// left edge the way the board's MARK column does.
func markLine(it markItem) string {
	var b strings.Builder

	switch store.Mark(it.Mark) {
	case store.MarkUp:
		b.WriteString("+  ")
	case store.MarkDown:
		b.WriteString("-  ")
	default:
		b.WriteString("?  ")
	}

	if it.ID > 0 {
		fmt.Fprintf(&b, "%4d  ", it.ID)
	} else {
		b.WriteString("   ?  ")
	}

	name := it.Title
	if name == "" {
		name = it.Slug + "  (not synced)"
	}
	fmt.Fprintf(&b, "%-44s", name)

	if it.Difficulty != "" {
		fmt.Fprintf(&b, "  %-6s", strings.ToLower(it.Difficulty))
	}
	if it.Status == "ac" {
		b.WriteString("  solved")
	}
	if it.Note != "" {
		fmt.Fprintf(&b, "  — %s", it.Note)
	}
	return strings.TrimRight(b.String(), " ")
}
