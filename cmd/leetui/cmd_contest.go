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
	"github.com/Nano-AI/leetui/internal/syncer"
)

// The contest subcommands (D-036).
//
// A contest is the one surface where the CLI matters more than the app: ninety minutes is
// not the time to learn a new screen, and `leetui contest pull` puts four folders on disk
// in one command so an editor can be open on the first problem before the timer starts.
//
//	leetui contest                    the schedule, with a countdown
//	leetui contest <slug>             one contest's problems
//	leetui contest pull <slug>        every problem's folder, laid out
//	leetui contest submit <slug> <p>  submit to the CONTEST judge, which is what scores
//
// `submit` is a separate verb from `leetui submit` on purpose and the reason is in
// leetcode.SubmitContest: the ordinary endpoint accepts a submission during a contest,
// judges it, and scores nothing. The two cannot be one command that guesses.

func runContest(a *app, args []string) (int, error) {
	fs, lang := flags("contest")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}
	for _, arg := range rest {
		if strings.TrimSpace(arg) == "" {
			return exitProblem, fmt.Errorf("contest and problem arguments must not be blank; omit the problem to use the current directory")
		}
	}

	switch verb := first(rest); verb {
	case "":
		return contestSchedule(a)
	case "pull":
		if len(rest) != 2 {
			return exitProblem, fmt.Errorf("usage: leetui contest pull <contest> [--lang language]")
		}
		return contestPull(a, argAt(rest, 1), *lang)
	case "submit":
		if len(rest) < 2 || len(rest) > 3 {
			return exitProblem, fmt.Errorf("usage: leetui contest submit <contest> [problem] [--lang language]")
		}
		return contestSubmit(a, argAt(rest, 1), argAt(rest, 2), *lang)
	default:
		if len(rest) != 1 {
			return exitProblem, fmt.Errorf("usage: leetui contest <contest>")
		}
		// Anything else is a contest slug. `leetui contest weekly-contest-517` is the
		// shape people reach for first, and making them type a verb for it would be
		// friction in the ninety minutes where it costs the most.
		return contestShow(a, verb)
	}
}

// argAt returns the nth argument, or "".
func argAt(args []string, n int) string {
	if n < len(args) {
		return args[n]
	}
	return ""
}

// contestSchedule prints the upcoming contests with a countdown to each.
func contestSchedule(a *app) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runSyncJob(ctx, func(out chan<- syncer.Progress) error {
		return a.sync.ContestSchedule(ctx, out)
	}); err != nil {
		// A refresh that fails is not fatal: the table may already hold the schedule,
		// and a cached answer beats no answer when the network is the problem.
		fmt.Fprintf(os.Stderr, "leetui: %v (showing what is cached)\n", err)
	}

	list, err := a.store.Contests(ctx, "")
	if err != nil {
		return exitProblem, err
	}
	if len(list) == 0 {
		return exitProblem, fmt.Errorf("no contests known; check the network and try again")
	}

	now := time.Now()
	for _, c := range list {
		fmt.Printf("%-24s  %-9s  %s  %s\n",
			c.Slug, c.PhaseAt(now), c.Start().Local().Format("Mon 02 Jan 15:04"),
			countdown(c, now))
	}
	return exitOK, nil
}

// countdown renders the phrase that belongs next to a contest in a list.
func countdown(c store.Contest, now time.Time) string {
	switch c.PhaseAt(now) {
	case leetcode.PhaseUpcoming:
		return "starts in " + short(c.Remaining(now))
	case leetcode.PhaseLive:
		return short(c.Remaining(now)) + " left"
	default:
		return ""
	}
}

// short formats a duration to the minute, which is the resolution a schedule needs.
//
// time.Duration's own String gives "1h23m45.6s", and the fractional seconds on a
// ninety-minute countdown are noise that changes every redraw.
func short(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// contestShow pulls one contest and lists its problems.
func contestShow(a *app, slug string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c, qs, err := pullContest(ctx, a, slug)
	if err != nil {
		return exitProblem, err
	}

	now := time.Now()
	fmt.Printf("%s  %s  %s\n", c.Title, c.PhaseAt(now), countdown(c, now))
	warnUnregistered(ctx, a, c)

	if len(qs) == 0 {
		// The normal state before the start, and worth saying plainly rather than
		// printing an empty list that reads as a failure.
		fmt.Fprintf(os.Stderr, "no problems yet — they open at %s\n",
			c.Start().Local().Format("15:04"))
		return exitOK, nil
	}
	for _, q := range qs {
		mark := " "
		if q.Solved {
			mark = "*"
		}
		fmt.Printf("%s %d pts  %-44s  %s\n", mark, q.Credit, q.Slug, q.Title)
	}
	return exitOK, nil
}

// contestPull lays out every problem in a contest.
//
// One command rather than four, because the first minute of a contest is the worst time
// to be typing slugs. A problem that fails to lay out does not stop the others: three
// folders on disk beats an error and none.
func contestPull(a *app, slug, langFlag string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	c, qs, err := pullContest(ctx, a, slug)
	if err != nil {
		return exitProblem, err
	}
	if len(qs) == 0 {
		return exitProblem, fmt.Errorf("%s has no problems yet — they open at %s",
			c.Slug, c.Start().Local().Format("15:04"))
	}

	warnUnregistered(ctx, a, c)

	l, err := a.language(langFlag, "")
	if err != nil {
		return exitProblem, err
	}

	failed := 0
	for _, q := range qs {
		// The contest-scoped fetch, not the ordinary one: while the contest runs its
		// problems are not in the problem set and the plain query answers null.
		d, err := a.sync.ContestDetail(ctx, c.Slug, q.Slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", q.Slug, err)
			failed++
			continue
		}
		out, err := solve.Prepare(a.cfg.Workspace, d, l)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", q.Slug, err)
			failed++
			continue
		}
		fmt.Printf("%d pts  %s\n", q.Credit, out.Solution)
	}
	if failed == len(qs) {
		// Every one failing means the statements are not readable yet, not that the
		// workspace is broken. During a live contest that is worth naming, because the
		// browser is right there and works.
		return exitProblem, fmt.Errorf("no problem in %s could be laid out — "+
			"the statements may not be readable from the API yet; "+
			"they are at https://leetcode.com/contest/%s/", c.Slug, c.Slug)
	}
	if failed > 0 {
		return exitProblem, fmt.Errorf("%d of %d could not be laid out; they are at %s",
			failed, len(qs), "https://leetcode.com/contest/"+c.Slug+"/")
	}
	return exitOK, nil
}

// contestSubmit sends one solution to the CONTEST judge.
//
// The problem argument may be omitted when run from inside a problem's folder, the same
// way `leetui submit` may be.
func contestSubmit(a *app, contest, problem, langFlag string) (int, error) {
	if contest == "" {
		return exitProblem, fmt.Errorf("usage: leetui contest submit <contest> [problem]")
	}
	if !a.client.Authenticated() {
		return exitProblem, fmt.Errorf("not signed in; open leetui and press a")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	slug, err := solve.Locate(problem)
	if err != nil {
		return exitProblem, err
	}

	// The question id comes from the contest table, not from problems. During a live
	// contest the problem is not in the problem set, so the contest's own record of its
	// internal id is the only place it exists on this machine.
	qs, err := a.store.ContestQuestions(ctx, contest)
	if err != nil {
		return exitProblem, err
	}
	var target store.ContestQuestion
	for _, q := range qs {
		if q.Slug == slug {
			target = q
			break
		}
	}
	if target.Slug == "" {
		return exitProblem, fmt.Errorf("%s is not a problem in %s; run `leetui contest %s` first",
			slug, contest, contest)
	}
	if target.QuestionID == "" {
		return exitProblem, fmt.Errorf("%s has no questionId cached; run `leetui contest %s` again",
			slug, contest)
	}

	d, err := a.sync.ContestDetail(ctx, contest, slug)
	if err != nil {
		return exitProblem, err
	}
	l, err := a.language(langFlag, problem)
	if err != nil {
		return exitProblem, err
	}
	out, err := solve.Select(a.cfg.Workspace, problem, d, l)
	if err != nil {
		return exitProblem, err
	}
	raw, err := os.ReadFile(out.Solution)
	if err != nil {
		return exitProblem, fmt.Errorf("read solution: %w", err)
	}
	code := runner.ExtractCode(l, string(raw))
	if strings.TrimSpace(code) == "" {
		return exitProblem, fmt.Errorf("nothing to submit: %s is empty between its markers", out.Solution)
	}

	fmt.Fprintf(os.Stderr, "submitting %s to %s as %s…\n", slug, contest, l.Display)

	id, err := a.client.SubmitContest(ctx, contest, leetcode.Submission{
		Slug: slug, QuestionID: target.QuestionID, Lang: l.Slug, Code: code,
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
		_ = a.store.SetContestSolved(ctx, contest, slug)
		commitAccepted(a, d, l, j, out)
	}
	return exit, nil
}

// warnUnregistered says so, loudly, when this account is not signed up for a contest it
// is about to be worked on.
//
// This is the highest-value thing the tool can say. An unregistered submission is judged,
// comes back Accepted, and scores nothing — the same silent failure the separate submit
// endpoint exists to prevent, one level up. LeetCode has no registration API, so this is
// checked and reported rather than fixed: the button is on the contest page.
//
// Silent when it cannot tell. Signed out the endpoint refuses, and an unanswerable
// question must not become a warning that cries wolf.
func warnUnregistered(ctx context.Context, a *app, c store.Contest) {
	if c.PhaseAt(time.Now()) == leetcode.PhaseEnded {
		return
	}

	// The session has to be confirmed live FIRST, because the contest endpoint reports
	// `registered: false` for an expired one instead of refusing it. Skipping this check
	// turns every stale cookie into a confident, wrong "not registered" — which is worse
	// than saying nothing, since the user goes and looks at a page that says Registered
	// and stops believing the tool.
	//
	// An expired session is also the more urgent of the two problems: nothing can be
	// submitted at all until it is fixed.
	st, err := a.client.Status(ctx)
	if err != nil && !errors.Is(err, leetcode.ErrSessionExpired) {
		return // A failed lookup cannot establish the account's state.
	}
	if errors.Is(err, leetcode.ErrSessionExpired) || !st.IsSignedIn {
		fmt.Fprintf(os.Stderr,
			"\n  NOT SIGNED IN — the stored session is expired.\n"+
				"  Nothing can be submitted until you sign in again: open leetui and press a.\n"+
				"  Registration cannot be checked until then.\n\n")
		return
	}

	info, err := a.client.Info(ctx, c.Slug)
	if err != nil || info.Registered {
		return
	}
	fmt.Fprintf(os.Stderr,
		"\n  NOT REGISTERED for %s (signed in as %s).\n"+
			"  Submissions will be judged and will score nothing.\n"+
			"  Register at https://leetcode.com/contest/%s/\n\n", c.Title, st.Username, c.Slug)
}

// pullContest refreshes one contest and returns it with its questions.
//
// Always refetches rather than trusting the table. During a live contest the question
// list is the thing that changes — it is empty until the start — so a cached empty answer
// is the one result that must never be believed.
func pullContest(ctx context.Context, a *app, slug string) (store.Contest, []store.ContestQuestion, error) {
	if slug == "" {
		return store.Contest{}, nil, fmt.Errorf("usage: leetui contest <contest>")
	}
	if err := runSyncJob(ctx, func(out chan<- syncer.Progress) error {
		return a.sync.Contest(ctx, slug, out)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "leetui: %v (showing what is cached)\n", err)
	}

	c, err := a.store.ContestOf(ctx, slug)
	if err != nil {
		return store.Contest{}, nil, fmt.Errorf("no contest called %q", slug)
	}
	qs, err := a.store.ContestQuestions(ctx, slug)
	if err != nil {
		return store.Contest{}, nil, err
	}
	return c, qs, nil
}

// runSyncJob drives a syncer job to completion, discarding its progress.
//
// The syncer reports through a channel because the TUI draws a bar from it. A subcommand
// has nothing to draw, but the channel still has to be drained or the job blocks on its
// first emit.
func runSyncJob(ctx context.Context, job func(chan<- syncer.Progress) error) error {
	ch := make(chan syncer.Progress, 8)
	done := make(chan error, 1)
	go func() { done <- job(ch) }()
	for range ch { //nolint:revive // draining is the point
	}
	return <-done
}
