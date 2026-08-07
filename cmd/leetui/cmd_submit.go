package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/vcs"
)

// runSubmit sends the solution to the judge and waits for the verdict.
//
// This creates a real submission on the user's account, so it does the least guessing of
// any command: an empty marked region, a missing session, or a missing questionId all
// stop before anything is sent.
func runSubmit(a *app, args []string) (int, error) {
	fs, lang := flags("submit")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	arg := first(rest)

	if !a.client.Authenticated() {
		return exitProblem, fmt.Errorf("not signed in; open leetui and press a")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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

	raw, err := os.ReadFile(out.Solution)
	if err != nil {
		return exitProblem, fmt.Errorf("read solution: %w", err)
	}
	// Only the marked region goes to the judge — the rest is local scaffolding LeetCode
	// supplies itself and would reject as a duplicate (D-014).
	code := runner.ExtractCode(l, string(raw))
	if strings.TrimSpace(code) == "" {
		return exitProblem, fmt.Errorf("nothing to submit: %s is empty between its markers", out.Solution)
	}

	// questionId, never the frontend id — they are different numbers and the judge takes
	// the internal one.
	if d.QuestionID == "" {
		return exitProblem, fmt.Errorf("%s has no questionId cached; run `leetui pull %s` first",
			d.Slug, d.Slug)
	}

	fmt.Fprintf(os.Stderr, "submitting %d. %s as %s…\n", d.NumericID, d.Title, l.Display)

	id, err := a.client.Submit(ctx, leetcode.Submission{
		Slug: d.Slug, QuestionID: d.QuestionID, Lang: l.Slug, Code: code,
	})
	if err != nil {
		return exitProblem, err
	}

	j, err := a.client.Poll(ctx, id, nil)
	if err != nil {
		return exitProblem, err
	}
	exit := reportJudgement(os.Stdout, j)

	if j.Accepted() {
		commitAccepted(a, d, l, j)
	}
	return exit, nil
}

// commitAccepted records an accepted solution in the workspace repository (D-011).
//
// Here as well as in the TUI so the two produce identical commits — an editor calling the
// CLI must not end up with a different history from the app.
//
// Best effort by design. The submission is the result; a repository that will not take a
// commit is worth a line on stderr and nothing more, and must never change the exit code
// an agent is branching on (docs/AGENTS.md).
func commitAccepted(a *app, d *store.Detail, l runner.Lang, j leetcode.Judgement) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := solve.Commit(ctx, a.cfg.Workspace, a.cfg.Git, solve.Solved{
		ID: d.NumericID, Slug: d.Slug, Title: d.Title,
		Lang: l.Display, Filename: l.Filename(),
		Runtime: j.Runtime, Percentile: j.RuntimePercentile,
	})
	switch {
	case errors.Is(err, vcs.ErrNotRepo), errors.Is(err, vcs.ErrNoGit),
		errors.Is(err, vcs.ErrNothingToCommit):
		// None of these is news. A workspace nobody has run `git init` in is the normal
		// state, and re-submitting an unchanged solution reaches "nothing to commit"
		// every time.
	case err != nil:
		fmt.Fprintf(os.Stderr, "not committed: %v\n", err)
	default:
		fmt.Fprintf(os.Stderr, "committed %s as %s\n", plural(res.Files, "file"), res.Short)
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
