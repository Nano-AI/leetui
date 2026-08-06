package tui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ready returns the selected problem, or a command explaining why it is not usable yet.
//
// Every solve action needs the same three things — a selected row, a cached statement,
// and a language the problem offers — so the check lives in one place and each action
// stays about its own job.
func (m Model) ready() (*store.Detail, runner.Lang, tea.Cmd) {
	if m.detail == nil || m.cursor >= len(m.rows) || m.detail.Slug != m.rows[m.cursor].Slug {
		return nil, m.lang, status("Still loading this problem.", false)
	}
	if len(m.detail.Snippets) == 0 {
		return nil, m.lang, status("No starter code cached yet. Give it a moment.", false)
	}
	if _, ok := m.detail.Snippets[m.lang.Slug]; !ok {
		return nil, m.lang, status(
			fmt.Sprintf("%s does not offer %s. Press l to pick another language.",
				m.detail.Title, m.lang.Display), true)
	}
	return m.detail, m.lang, nil
}

// startEdit lays out the problem folder and opens the solution in the editor.
//
// With no editor chosen yet, this opens the picker instead of guessing. Guessing is how
// someone ends up in vi without knowing how to leave.
func (m Model) startEdit() (tea.Model, tea.Cmd) {
	d, lang, bail := m.ready()
	if bail != nil {
		return m, bail
	}
	if m.cfg.Editor == "" {
		return m.openEditorPicker()
	}
	m.enterProblem()
	return m, m.editCmd(d, lang)
}

// openEditorPicker lists the editors installed on this machine.
func (m Model) openEditorPicker() (tea.Model, tea.Cmd) {
	m.editors = editor.Detect()
	if len(m.editors) == 0 {
		return m, status("No editor found. Set one in "+m.cfg.Path()+".", true)
	}
	m.picking, m.pickIdx = pickEditor, 0
	return m, nil
}

// startRun executes the solution locally.
//
// Languages with no local driver are routed to the judge rather than refused — that is
// a routing decision, not a failure (D-004).
func (m Model) startRun() (tea.Model, tea.Cmd) {
	d, lang, bail := m.ready()
	if bail != nil {
		return m, bail
	}
	if m.running {
		return m, status("Already running.", false)
	}

	if !m.engine.Supports(lang) {
		hint := fmt.Sprintf("%s has no local runner. Press s to submit instead.", lang.Display)
		if local, ok := m.engine.(*runner.Local); ok {
			if bin := local.MissingToolchain(lang); bin != "" {
				hint = fmt.Sprintf("%s needs %s on your PATH. Install it, or press s to submit.",
					lang.Display, bin)
			}
		}
		return m, status(hint, true)
	}

	m.enterProblem()
	m.running = true
	return m, tea.Batch(m.runLocalCmd(d, lang), status("Running "+lang.Display+"…", false))
}

// startSubmit sends the solution to the judge and puts a flap in the queue.
func (m Model) startSubmit() (tea.Model, tea.Cmd) {
	d, lang, bail := m.ready()
	if bail != nil {
		return m, bail
	}
	if !m.client.Authenticated() {
		return m, status("Sign in first — press a.", true)
	}

	m.enterProblem()

	flap := components.NewFlap(m.nextFlapID, theme.Display(theme.Pending.Text()), theme.Amber)
	m.nextFlapID++

	m.queue = append([]queueItem{{
		ProblemID: d.NumericID, Lang: lang.Slug, Verdict: theme.Pending, flap: flap,
	}}, m.queue...)
	if len(m.queue) > 8 {
		m.queue = m.queue[:8]
	}

	return m, tea.Batch(
		m.submitCmd(d, lang, flap.ID),
		status("Submitted "+d.Title+" as "+lang.Display+"…", false),
	)
}

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
		return m, tea.Batch(cmd, status(judgeSummary(msg.judgement), !msg.judgement.Accepted()))
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
