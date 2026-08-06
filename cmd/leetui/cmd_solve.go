package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
)

// runPull lays out a problem's folder and prints where it went.
//
// Printing the path is the point as much as the files are: it is what lets a shell
// function cd into it, or an editor open it.
func runPull(a *app, args []string) (int, error) {
	fs, lang := flags("pull")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	arg := first(rest)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	d, err := a.problem(ctx, arg)
	if err != nil {
		return exitProblem, err
	}
	l, err := a.language(*lang, arg)
	if err != nil {
		return exitProblem, err
	}

	out, err := solve.Prepare(a.cfg.Workspace, d, l)
	if err != nil {
		return exitProblem, err
	}
	fmt.Printf("%d. %s  %s\n", d.NumericID, d.Title, d.Difficulty)
	fmt.Println(out.Solution)
	return exitOK, nil
}

// runPath prints a problem's folder without touching anything.
func runPath(a *app, args []string) (int, error) {
	fs, _ := flags("path")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	d, err := a.problem(ctx, first(rest))
	if err != nil {
		return exitProblem, err
	}
	dir, err := solve.Dir(a.cfg.Workspace, d)
	if err != nil {
		return exitProblem, err
	}
	fmt.Println(dir)
	return exitOK, nil
}

// runRun executes the solution locally and reports each case.
//
// It prepares first, so `leetui run two-sum` works on a machine that has never opened the
// problem — the folder, the scaffolded solution, and the test cases all appear.
func runRun(a *app, args []string) (int, error) {
	fs, lang := flags("run")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	arg := first(rest)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	d, err := a.problem(ctx, arg)
	if err != nil {
		return exitProblem, err
	}
	l, err := a.language(*lang, arg)
	if err != nil {
		return exitProblem, err
	}

	engine := runner.NewLocal()
	if !engine.Supports(l) {
		if bin := engine.MissingToolchain(l); bin != "" {
			return exitProblem, fmt.Errorf("%s needs %s on your PATH", l.Display, bin)
		}
		return exitProblem, fmt.Errorf("%s has no local runner; use `leetui submit`", l.Display)
	}

	out, err := solve.Prepare(a.cfg.Workspace, d, l)
	if err != nil {
		return exitProblem, err
	}

	res, err := solve.Run(ctx, engine, out.Dir, d, l)
	if err != nil {
		return exitProblem, err
	}
	return reportRun(os.Stdout, d.Slug, res), nil
}

// first returns the first positional argument, or "" for none.
func first(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
