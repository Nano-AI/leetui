package main

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
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
		name: "contest", usage: "[slug|pull|submit] …",
		summary: "the contest schedule, its problems, and the judge that scores them",
		run:     runContest,
	},
	{
		name: "todo", usage: "[add|rm|list] …",
		summary: "the list of problems you mean to get to",
		run:     runTodo,
	},
	{
		name: "mark", usage: "[up|down|clear]",
		summary: "record which problems were worth doing, for a second pass",
		run:     runMark,
	},
	{
		name: "path", usage: "[problem]",
		summary: "print a problem's folder, for scripting",
		run:     runPath,
	},
	{
		name: "image", usage: "[problem] [n]",
		summary: "draw a statement's figure, on a terminal that can show one",
		run:     runImage,
	},
	{
		name:    "doctor",
		summary: "check this machine: toolchains, editor, terminal images",
		run:     runDoctor,
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
	fmt.Fprintln(w, "The todo list and the marks are meant to be driven by scripts and agents:")
	fmt.Fprintln(w, "  leetui todo add two-sum --note \"from the JD\"")
	fmt.Fprintln(w, "  leetui todo --json")
	fmt.Fprintln(w, "  leetui mark up two-sum --note \"worth redoing\"")
	fmt.Fprintln(w, "  leetui mark --json --up")
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
// Parse one flag (and its value) at a time so -- terminates parsing for the whole
// command, while a literal -- used as a string flag's value remains a value.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for len(args) > 0 {
		arg := args[0]
		if arg == "--" {
			return append(positional, args[1:]...), nil
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			args = args[1:]
			continue
		}
		n := 1
		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		if !strings.Contains(name, "=") {
			if f := fs.Lookup(name); f != nil {
				b, ok := f.Value.(interface{ IsBoolFlag() bool })
				if (!ok || !b.IsBoolFlag()) && len(args) > 1 {
					n = 2
				}
			}
		}
		if err := fs.Parse(args[:n]); err != nil {
			return nil, err
		}
		args = args[n:]
	}
	return positional, nil
}
