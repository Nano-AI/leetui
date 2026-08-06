package main

import (
	"flag"
	"fmt"
	"io"
	"sort"
)

// The command-line surface.
//
// leetui with no arguments is the TUI, which is the product. The subcommands are a SEAM,
// not a second interface: they exist so an editor can drive the same core without leetui
// having to grow a plugin for every editor that exists (D-015).
//
// The shapes that matter are the ones an editor produces:
//
//	:!leetui run %          the buffer's path
//	leetui run              from inside the problem folder
//	leetui run two-sum      the slug, from anywhere
//
// Exit codes are part of the contract, because that is what an editor branches on:
//
//	0  it worked, and for `run` every case passed
//	1  it ran and the answer was wrong
//	2  it could not run at all

const (
	exitOK      = 0
	exitFailed  = 1
	exitProblem = 2
)

// command is one subcommand.
type command struct {
	name    string
	summary string
	// usage is the argument shape, shown after the name.
	usage string
	run   func(a *app, args []string) (int, error)
}

var commands = []command{
	{
		name: "pull", usage: "[problem]",
		summary: "lay out a problem's folder: statement, solution, test cases",
		run:     runPull,
	},
	{
		name: "run", usage: "[problem|file]",
		summary: "run the solution locally against its test cases",
		run:     runRun,
	},
	{
		name: "submit", usage: "[problem|file]",
		summary: "submit the solution to the judge and wait for a verdict",
		run:     runSubmit,
	},
	{
		name: "todo", usage: "[add|rm|list] …",
		summary: "the list of problems you mean to get to",
		run:     runTodo,
	},
	{
		name: "path", usage: "[problem]",
		summary: "print a problem's folder, for scripting",
		run:     runPath,
	},
}

func lookupCommand(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

// usage explains the whole surface, including that no arguments opens the TUI.
func usage(w io.Writer) {
	fmt.Fprintln(w, "leetui — a terminal client for LeetCode")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  leetui                     open the app")
	fmt.Fprintln(w)

	sorted := append([]command(nil), commands...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })

	for _, c := range sorted {
		fmt.Fprintf(w, "  leetui %-8s %-14s %s\n", c.name, c.usage, c.summary)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "A problem can be a slug (two-sum), a folder (0001-two-sum), a path to")
	fmt.Fprintln(w, "either, or omitted to mean the current directory.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "From an editor, the file you are looking at is enough:")
	fmt.Fprintln(w, "  :!leetui run %")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The todo list is meant to be driven by scripts and agents:")
	fmt.Fprintln(w, "  leetui todo add two-sum --note \"from the JD\"")
	fmt.Fprintln(w, "  leetui todo --json")
	fmt.Fprintln(w, "See docs/AGENTS.md.")
}

// flags builds a flag set that reports errors through the command's own usage rather
// than the global one.
func flags(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("leetui "+name, flag.ContinueOnError)
	lang := fs.String("lang", "",
		"language slug, e.g. python3. Defaults to the file's, then to default_lang.")
	return fs, lang
}

// parseFlags parses a flag set allowing flags ANYWHERE among the arguments.
//
// Go's flag package stops at the first non-flag, so `leetui todo add two-sum --note x`
// silently treats `--note` as another problem name. That is the order a person writes
// naturally and the order an agent will generate, so it has to work.
//
// The loop is the standard interspersed-parsing idiom: parse, take one positional, parse
// what is left, repeat.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		rest = fs.Args()[1:]
	}
}
