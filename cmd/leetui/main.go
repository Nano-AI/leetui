// Command leetui is a terminal client for LeetCode.
//
// With no arguments it opens the app. The subcommands are a seam for editors and scripts
// over the same core — see cli.go and D-015.
//
// See docs/DECISIONS.md for the architecture record, docs/DESIGN.md for the visual
// system, and docs/ROADMAP.md for what is built so far.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Nano-AI/leetui/internal/solve"
)

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// dispatch routes to a subcommand or to the TUI.
//
// Bare flags still belong to the TUI, so `leetui --no-motion` keeps working. Only a
// leading bare word is treated as a subcommand — which also means a future flag can never
// be mistaken for one.
func dispatch(args []string) int {
	if len(args) == 0 || args[0][0] == '-' {
		switch args := args; {
		case len(args) == 1 && (args[0] == "-h" || args[0] == "--help"):
			usage(os.Stdout)
			return exitOK
		}
		if err := runTUI(args); err != nil {
			fmt.Fprintf(os.Stderr, "leetui: %v\n", err)
			return exitProblem
		}
		return exitOK
	}

	name := args[0]
	if name == "help" {
		usage(os.Stdout)
		return exitOK
	}

	cmd, ok := lookupCommand(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "leetui: unknown command %q\n\n", name)
		usage(os.Stderr)
		return exitProblem
	}

	a, err := open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "leetui: %v\n", err)
		return exitProblem
	}
	defer a.Close()

	code, err := cmd.run(a, args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "leetui: %v\n", err)
		if errors.Is(err, solve.ErrNoProblem) {
			fmt.Fprintf(os.Stderr,
				"\nName a problem, or run this from inside its folder:\n"+
					"  leetui %s two-sum\n", name)
		}
		return code
	}
	return code
}
