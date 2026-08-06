package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
)

// Reporting a run on a terminal that is not the TUI.
//
// Plain text, no colour, no box drawing. This output lands in an editor's message area or
// a scrollback buffer as often as in a terminal, and neither renders a Departure Board.
//
// A failing case prints input, expected, and actual stacked, for the same reason the TUI's
// diff panel does: the three together are the whole diagnosis, and a bare "FAIL" sends you
// looking for them.

// reportRun writes a local run and returns the process exit code.
func reportRun(w io.Writer, slug string, r runner.Result) int {
	if r.CompileErr != "" {
		fmt.Fprintln(w, "did not compile:")
		fmt.Fprintln(w, indent(r.CompileErr))
		return exitProblem
	}

	passed, failed, errored := r.Summary()

	for i, c := range r.Cases {
		switch {
		case c.Err != nil:
			fmt.Fprintf(w, "case %d  ERROR\n%s\n", i+1, indent(c.Err.Error()))
		case !c.Judged:
			fmt.Fprintf(w, "case %d  ran    %s\n", i+1, oneLine(c.Actual))
		case c.Passed:
			fmt.Fprintf(w, "case %d  ok     %s\n", i+1, oneLine(c.Actual))
		default:
			fmt.Fprintf(w, "case %d  FAIL\n", i+1)
			fmt.Fprintf(w, "  in    %s\n", oneLine(c.Case.Input))
			fmt.Fprintf(w, "  want  %s\n", oneLine(c.Case.Expected))
			fmt.Fprintf(w, "  got   %s\n", oneLine(c.Actual))
		}
	}

	switch {
	case errored > 0:
		fmt.Fprintf(w, "\n%d of %d cases crashed.\n", errored, len(r.Cases))
		return exitFailed
	case failed > 0 && !runner.HasOverride(slug):
		// A local mismatch never claims authority: metaData cannot express in-place,
		// unordered, or float-tolerant answers (D-003).
		fmt.Fprintf(w, "\n%d of %d cases mismatched. This problem has no curated comparator,\n"+
			"so check it on the judge: leetui submit %s\n", failed, passed+failed, slug)
		return exitFailed
	case failed > 0:
		fmt.Fprintf(w, "\n%d of %d cases failed.\n", failed, passed+failed)
		return exitFailed
	case passed > 0:
		fmt.Fprintf(w, "\nAll %d cases passed. Submit with: leetui submit %s\n", passed, slug)
		return exitOK
	default:
		fmt.Fprintf(w, "\nRan %d cases; none had an expected answer to check against.\n", len(r.Cases))
		return exitOK
	}
}

// reportJudgement writes a submission's verdict and returns the process exit code.
func reportJudgement(w io.Writer, j leetcode.Judgement) int {
	fmt.Fprintln(w, j.StatusMsg)

	if j.Accepted() {
		if j.Runtime != "" {
			fmt.Fprintf(w, "runtime %s", j.Runtime)
			if j.RuntimePercentile > 0 {
				fmt.Fprintf(w, " (beats %.1f%%)", j.RuntimePercentile)
			}
			fmt.Fprintln(w)
		}
		if j.Memory != "" {
			fmt.Fprintf(w, "memory  %s", j.Memory)
			if j.MemoryPercentile > 0 {
				fmt.Fprintf(w, " (beats %.1f%%)", j.MemoryPercentile)
			}
			fmt.Fprintln(w)
		}
		return exitOK
	}

	if j.TotalTestcases > 0 {
		fmt.Fprintf(w, "passed %d of %d cases\n", j.TotalCorrect, j.TotalTestcases)
	}
	if j.LastTestcase != "" {
		fmt.Fprintf(w, "  in    %s\n", oneLine(j.LastTestcase))
		fmt.Fprintf(w, "  want  %s\n", oneLine(j.ExpectedOutput))
		fmt.Fprintf(w, "  got   %s\n", oneLine(strings.Join(j.CodeOutput, "\n")))
	}
	// Fullest message first: the short form is a summary of the long one.
	for _, msg := range []string{j.FullCompileError, j.CompileError, j.RuntimeError} {
		if strings.TrimSpace(msg) != "" {
			fmt.Fprintln(w, indent(msg))
			break
		}
	}
	return exitFailed
}

// oneLine flattens a value so a multi-parameter input stays on the line that labels it.
func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(empty)"
	}
	return strings.ReplaceAll(s, "\n", " · ")
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
