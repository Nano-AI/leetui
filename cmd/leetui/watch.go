package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/store"
)

// `leetui run --watch` — the results pane, as its own process.
//
// The app already re-runs on save, but its result panel lives in a 40%-wide column beside
// the statement, and a compiler error wants the whole terminal. Rather than have leetui
// arrange windows — that road ends with leetui reimplementing tmux, which is the same
// argument that keeps the editor out of it (D-012) — this makes the results a plain
// command you can put wherever you like:
//
//	┌───────────────┬──────────────────┐
//	│  leetui       │  nvim            │
//	│  (statement)  ├──────────────────┤
//	│               │  leetui run -w   │
//	└───────────────┴──────────────────┘
//
// Three ordinary processes. tmux does the layout, the user picks the proportions, and
// nothing here knows or cares about any of it.

// watchInterval is how often the solution file is checked.
//
// Polling rather than fsnotify, matching the TUI's watcher: one stat call is cheaper than
// a dependency, and a re-run that starts 400ms after a save is indistinguishable from one
// that starts instantly.
const watchInterval = 400 * time.Millisecond

// runWatch re-runs a problem's tests every time its solution file changes.
func runWatch(a *app, arg, langFlag string) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, err := a.problem(ctx, arg)
	if err != nil {
		return exitProblem, err
	}
	l, err := a.language(langFlag, arg)
	if err != nil {
		return exitProblem, err
	}

	engine := runner.NewLocal()
	if !engine.Supports(l) {
		return exitProblem, fmt.Errorf("%s has no local runner; --watch has nothing to do", l.Display)
	}

	out, err := solve.Prepare(a.cfg.Workspace, d, l)
	if err != nil {
		return exitProblem, err
	}

	fmt.Fprintf(os.Stderr, "watching %s\n", out.Solution)
	fmt.Fprintf(os.Stderr, "%d. %s  %s  ·  ctrl-c to stop\n\n", d.NumericID, d.Title, l.Display)

	// The first sighting only records the time. Opening a solution written last week
	// should not fire a run before the user has typed anything.
	last := modTime(out.Solution)
	runOnce(ctx, a, engine, out.Dir, d, l)

	for {
		select {
		case <-ctx.Done():
			return exitOK, nil
		case <-time.After(watchInterval):
		}

		now := modTime(out.Solution)
		if now.Equal(last) {
			continue
		}
		last = now

		// Clearing keeps the pane showing ONE result. A scrollback of eleven previous
		// attempts buries the current one, which is the only one being asked about.
		clearScreen(os.Stdout)
		fmt.Fprintf(os.Stderr, "%s  %s\n\n", d.Title, time.Now().Format("15:04:05"))
		runOnce(ctx, a, engine, out.Dir, d, l)
	}
}

// runOnce executes the tests and prints them, reporting a failure rather than returning
// it: a watch loop that exits on the first wrong answer would be useless.
func runOnce(ctx context.Context, a *app, engine runner.Engine, dir string, d *store.Detail, l runner.Lang) {
	res, err := solve.Run(ctx, engine, dir, d, l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not run: %v\n", err)
		return
	}
	reportRun(os.Stdout, d.Slug, res)
}

func modTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// clearScreen resets the terminal between runs.
//
// Written directly rather than shelling to `clear`: this is two bytes of ANSI that every
// terminal has understood for forty years, and spawning a process for it would be silly.
func clearScreen(w io.Writer) {
	fmt.Fprint(w, "\x1b[2J\x1b[H")
}
