package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSolveMsg routes the solve-loop messages: editing, local runs, and judgements.
//
// Update dispatches here so its own switch stays a routing table rather than growing
// into a wall of unrelated logic.
func (m Model) handleSolveMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case editReadyMsg:
		// Hand the terminal to the editor and take it back cleanly. Bubbletea restores
		// the alt screen and mouse mode around the child process; leaving either on
		// while a full-screen editor runs corrupts both.
		//
		// Arguments go as a slice, never a shell string — the path is built from a
		// LeetCode slug.
		cmd := exec.Command(msg.editor, msg.file)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editDoneMsg{err: err}
		})

	case editDoneMsg:
		if msg.err != nil {
			return m, status("Editor exited with an error: "+msg.err.Error(), true)
		}
		// The file may have changed, so any previous run result is now stale.
		m.runResult, m.runSlug = nil, ""
		return m, nil

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
