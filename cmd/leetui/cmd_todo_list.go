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

// Reading the list out, for a person and for a program.
//
// The JSON shape is documented as stable in docs/AGENTS.md — agents parse it, so changing
// a field name is a breaking change and belongs in that file first.

func todoList(a *app, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui todo list", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print the list as JSON, for scripts and agents")
	if _, err := parseFlags(fs, args); err != nil {
		return exitProblem, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := a.store.Todos(ctx)
	if err != nil {
		return exitProblem, err
	}

	items := make([]todoItem, 0, len(entries))
	for _, e := range entries {
		it := todoItem{Slug: e.Slug, Note: e.Note, AddedAt: e.AddedAt.UTC().Format(time.RFC3339)}
		// An entry may name a problem this machine has not synced; the list still holds,
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
		// Always an array, never null: a script that pipes this into a loop should not
		// have to special-case an empty list.
		return exitOK, enc.Encode(items)
	}

	if len(items) == 0 {
		fmt.Println("Nothing on the list. Add one: leetui todo add two-sum")
		return exitOK, nil
	}
	for _, it := range items {
		fmt.Printf("%s\n", todoLine(it))
	}
	return exitOK, nil
}

// todoLine is one entry as a human reads it.
func todoLine(it todoItem) string {
	var b strings.Builder
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

func problemURL(d *store.Detail) string {
	return "https://leetcode.com/problems/" + d.Slug + "/"
}
