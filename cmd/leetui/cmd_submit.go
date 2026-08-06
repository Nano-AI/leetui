package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
)

// runSubmit sends the solution to the judge and waits for the verdict.
//
// This creates a real submission on the user's account, so it does the least guessing of
// any command: an empty marked region, a missing session, or a missing questionId all
// stop before anything is sent.
func runSubmit(a *app, args []string) (int, error) {
	fs, lang := flags("submit")
	if err := fs.Parse(args); err != nil {
		return exitProblem, err
	}
	arg := fs.Arg(0)

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
	return reportJudgement(os.Stdout, j), nil
}
