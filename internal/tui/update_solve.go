package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleSolveMsg routes the solve-loop messages: editing, local runs, and judgements.
//
// Update dispatches here so its own switch stays a routing table rather than growing
// into a wall of unrelated logic.
func (m Model) handleSolveMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case editReadyMsg:
		return m.launchEditor(msg.plan)

	case editDoneMsg:
		if msg.err != nil {
			return m, status("Editor exited with an error: "+msg.err.Error(), true)
		}
		// The file may have changed, so any previous run result is now stale.
		m.runResult, m.runSlug = nil, ""

		// Run the tests now rather than making the user press r.
		//
		// This path is the one where leetui was suspended, so the watcher saw nothing
		// while the editing happened. Recording the new timestamp first is what stops
		// the next watch tick firing a second, identical run.
		if !m.cfg.RunAfterEdit || !m.noteSolutionChanged() {
			return m, nil
		}
		return m.startRun()

	case runFinishedMsg:
		m.running = false
		if msg.err != nil {
			return m, status(runErrorHint(msg.err), true)
		}
		m.runResult, m.runSlug = &msg.result, msg.slug
		return m, status(runSummary(msg.slug, msg.result), !msg.result.Passed())

	case judgeMsg:
		return m.handleJudgement(msg)
	}

	return m, nil
}
