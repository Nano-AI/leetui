package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/store"
)

// The todo list, from the command line.
//
// This exists to be driven by something other than a human: an agent that has just read a
// job description, or a script working through a syllabus, can queue problems here and
// leetui shows them at the top of the board next time it opens.
//
// So `--json` is a first-class output rather than an afterthought, and every subcommand
// is idempotent — adding something twice is not an error, and neither is removing
// something absent. A tool that has to check before it acts is a tool that races.

// runTodo dispatches the todo subcommands.
func runTodo(a *app, args []string) (int, error) {
	if len(args) == 0 {
		return todoList(a, nil)
	}
	switch args[0] {
	case "add":
		return todoAdd(a, args[1:])
	case "rm", "remove", "done":
		return todoRemove(a, args[1:])
	case "list", "ls":
		return todoList(a, args[1:])
	case "clear":
		return todoClear(a, args[1:])
	default:
		// No subcommand: treat it as flags to `list`, so `leetui todo --json` works.
		if strings.HasPrefix(args[0], "-") {
			return todoList(a, args)
		}
		return exitProblem, fmt.Errorf("unknown todo command %q; try add, rm, list, or clear", args[0])
	}
}

// todoItem is the JSON shape. Stable: agents parse this.
type todoItem struct {
	Slug       string `json:"slug"`
	Title      string `json:"title,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
	ID         int    `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	Note       string `json:"note,omitempty"`
	AddedAt    string `json:"added_at"`
	URL        string `json:"url,omitempty"`
}

func todoAdd(a *app, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui todo add", flag.ContinueOnError)
	note := fs.String("note", "", "why this is on the list")
	problems, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	if len(problems) == 0 {
		return exitProblem, fmt.Errorf("name at least one problem: leetui todo add two-sum")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, arg := range problems {
		// Resolved through the store so a typo is caught here rather than sitting on the
		// list as a slug that will never match anything.
		d, err := a.problem(ctx, arg)
		if err != nil {
			return exitProblem, err
		}
		if err := a.store.AddTodo(ctx, d.Slug, *note); err != nil {
			return exitProblem, err
		}
		fmt.Printf("added %d. %s\n", d.NumericID, d.Title)
	}
	return exitOK, nil
}

func todoRemove(a *app, args []string) (int, error) {
	if len(args) == 0 {
		return exitProblem, fmt.Errorf("name at least one problem: leetui todo rm two-sum")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, arg := range args {
		// Resolve locally only. Removing must work for something the problem list no
		// longer has, and must not need the network to take an item off a list.
		slug, err := solve.Locate(arg)
		if err != nil {
			return exitProblem, err
		}
		listed, err := a.store.IsTodo(ctx, slug)
		if err != nil {
			return exitProblem, err
		}
		if err := a.store.RemoveTodo(ctx, slug); err != nil {
			return exitProblem, err
		}
		// Removing something absent is still a success — the caller wanted it gone and it
		// is — but claiming to have removed it would be a lie.
		if listed {
			fmt.Printf("removed %s\n", slug)
		} else {
			fmt.Printf("%s was not on the list\n", slug)
		}
	}
	return exitOK, nil
}

func todoClear(a *app, args []string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := a.store.Todos(ctx)
	if err != nil {
		return exitProblem, err
	}
	for _, e := range entries {
		if err := a.store.RemoveTodo(ctx, e.Slug); err != nil {
			return exitProblem, err
		}
	}
	fmt.Printf("cleared %d\n", len(entries))
	return exitOK, nil
}

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
