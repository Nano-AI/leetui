package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/store"
)

// Importance marks, from the command line (D-032).
//
// This is the half of the feature meant for an agent. A model that has just watched you
// work through Top Interview 150 knows which problems taught you something and which were
// a rerun of the one before; this is where it writes that down, and `leetui mark --json`
// is how it reads back what it already decided.
//
// Same contract as `todo` (D-022): every operation is idempotent, `--json` is always an
// array and never null, and the shape is documented as stable in docs/AGENTS.md.

// runMark dispatches the mark subcommands.
func runMark(a *app, args []string) (int, error) {
	if len(args) == 0 {
		return markList(a, nil)
	}
	switch args[0] {
	case "up", "important", "+":
		return markSet(a, store.MarkUp, args[1:])
	case "down", "unimportant", "-":
		return markSet(a, store.MarkDown, args[1:])
	case "clear", "rm", "remove":
		return markClear(a, args[1:])
	case "list", "ls":
		return markList(a, args[1:])
	default:
		// No subcommand: treat it as flags to `list`, so `leetui mark --json` works.
		if strings.HasPrefix(args[0], "-") && !isDirection(args[0]) {
			return markList(a, args)
		}
		return exitProblem, fmt.Errorf(
			"unknown mark command %q; try up, down, clear, or list", args[0])
	}
}

// isDirection keeps a bare "-" readable as "mark down" rather than as a malformed flag.
// "+" needs no such care; Go's flag package has no opinion about it.
func isDirection(arg string) bool { return arg == "-" }

// markItem is the JSON shape. Stable: agents parse this.
//
// `mark` is the word, not a glyph or a number — "up" survives a JSON round trip, a jq
// filter, and a human reading a log, which "+1" does less well.
type markItem struct {
	Slug       string `json:"slug"`
	Mark       string `json:"mark"`
	Title      string `json:"title,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
	ID         int    `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	Note       string `json:"note,omitempty"`
	MarkedAt   string `json:"marked_at"`
	URL        string `json:"url,omitempty"`
}

func markSet(a *app, mark store.Mark, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui mark "+string(mark), flag.ContinueOnError)
	note := fs.String("note", "", "why it earned this verdict")
	problems, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	if len(problems) == 0 {
		return exitProblem, fmt.Errorf(
			"name at least one problem: leetui mark %s two-sum", mark)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, arg := range problems {
		// Resolved through the store so a typo is caught here rather than sitting in the
		// marks table as a slug that will never match a row.
		d, err := a.problem(ctx, arg)
		if err != nil {
			return exitProblem, err
		}
		if err := a.store.SetMark(ctx, d.Slug, mark, *note); err != nil {
			return exitProblem, err
		}
		fmt.Printf("%s %d. %s\n", mark.Label(), d.NumericID, d.Title)
	}
	return exitOK, nil
}

func markClear(a *app, args []string) (int, error) {
	if len(args) == 0 {
		return exitProblem, fmt.Errorf("name at least one problem: leetui mark clear two-sum")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, arg := range args {
		// Resolve locally only. Clearing must work for a problem the list no longer has,
		// and must never need the network to withdraw an opinion.
		slug, err := solve.Locate(arg)
		if err != nil {
			return exitProblem, err
		}
		had, err := a.store.MarkOf(ctx, slug)
		if err != nil {
			return exitProblem, err
		}
		if err := a.store.ClearMark(ctx, slug); err != nil {
			return exitProblem, err
		}
		// Clearing something unmarked is still a success — the caller wanted no opinion
		// recorded, and there is none — but claiming to have cleared one would be a lie.
		if had == store.MarkNone {
			fmt.Printf("%s was not marked\n", slug)
		} else {
			fmt.Printf("cleared %s\n", slug)
		}
	}
	return exitOK, nil
}
