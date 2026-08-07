package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// What comes back: the judge's verdict, and a local run's result.
//
// Both end in words the user reads, and both are careful about how much authority they
// claim. The judge is the record; a local run is a fast approximation that says so.

// handleJudgement flips the queue row to the judge's verdict.
func (m Model) handleJudgement(msg judgeMsg) (tea.Model, tea.Cmd) {
	for i := range m.queue {
		if m.queue[i].flap.ID != msg.flapID {
			continue
		}
		if msg.err != nil {
			m.queue[i].Verdict = theme.RuntimeError
			cmd := m.queue[i].flap.FlipTo(theme.Display("failed"), theme.WA)
			return m, tea.Batch(cmd, status(judgeErrorHint(msg.err), true))
		}

		v := verdictOf(msg.judgement)
		m.queue[i].Verdict = v
		cmd := m.queue[i].flap.FlipTo(theme.Display(v.Text()), v.Color())

		// Record the verdict locally. Without this the board keeps whatever the last full
		// sync said — you submit, the judge says Accepted, and the row still reads TRIED
		// until the next re-sync.
		cmds := []tea.Cmd{cmd, m.recordVerdict(msg.judgement),
			status(judgeSummary(msg.judgement), !msg.judgement.Accepted())}

		// Commit the moment the work is provably correct (D-011). Only on Accepted: a
		// wrong answer is not a milestone. Returns nil when the feature is off, and says
		// nothing when the workspace is not a repository — see handleCommitted.
		if msg.judgement.Accepted() {
			cmds = append(cmds, m.commitAccepted(msg.judgement))
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// runSummary phrases a local result.
//
// A local failure never claims authority: metaData cannot express in-place, unordered,
// or float-tolerant answers, so an uncurated mismatch might be the comparator's fault
// rather than the solution's (D-003).
func runSummary(slug string, r runner.Result) string {
	passed, failed, errored := r.Summary()

	switch {
	case r.CompileErr != "":
		return "Did not compile. Press s to see the judge's message."
	case errored > 0:
		return fmt.Sprintf("%d of %d cases crashed.", errored, len(r.Cases))
	case failed == 0 && passed > 0:
		return fmt.Sprintf("All %d cases passed. Press s to submit.", passed)
	case failed > 0 && !runner.HasOverride(slug):
		return fmt.Sprintf("%d of %d cases mismatched — this problem has no curated "+
			"comparator, so press s to check on the judge.", failed, passed+failed)
	case failed > 0:
		return fmt.Sprintf("%d of %d cases failed.", failed, passed+failed)
	default:
		return fmt.Sprintf("Ran %d cases; none had an expected answer to check against.", len(r.Cases))
	}
}

func runErrorHint(err error) string {
	switch {
	case errors.Is(err, runner.ErrNoToolchain):
		return err.Error()
	case errors.Is(err, runner.ErrLangNotLocal):
		return err.Error() + ". Press s to run it on the judge."
	default:
		return "Could not run: " + err.Error()
	}
}

// recordVerdict writes a judgement into the local store and refreshes the board.
//
// The slug comes from the submission that is being judged, not from the cursor: judging
// is asynchronous, and by the time a verdict lands the user may well have moved on to
// another problem.
func (m Model) recordVerdict(j leetcode.Judgement) tea.Cmd {
	slug := m.runSlug
	if m.detail != nil {
		slug = m.detail.Slug
	}
	if slug == "" {
		return nil
	}

	st := leetcode.StatusAttempted
	if j.Accepted() {
		st = leetcode.StatusAccepted
	}
	store := m.store
	reload := m.loadRows()

	return tea.Sequence(
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = store.SetStatus(ctx, slug, st)
			return nil
		},
		reload,
	)
}
