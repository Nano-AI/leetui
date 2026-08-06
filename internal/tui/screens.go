package tui

import tea "github.com/charmbracelet/bubbletea"

// Moving between the two screens (D-018).
//
// The list and the problem want opposite things from the terminal. A list wants every
// column it can get and no distractions; a statement wants to be the only thing you are
// reading. leetcode.com splits them for the same reason, and so does this.
//
// enter opens, esc closes. Nothing else navigates between them, and the cursor never
// moves as a side effect — coming back to the list must put you exactly where you left,
// or browsing becomes a game of finding your place again.

// openProblem shows the selected problem's own screen.
func (m Model) openProblem() (tea.Model, tea.Cmd) {
	if m.currentSlug() == "" {
		return m, nil
	}
	m.mode = modeSolve
	m.focus = paneDetail
	m.detailScroll = 0

	// The statement may not have been fetched: on the list it was never needed, because
	// the list never showed it. Ask now.
	return m, m.loadDetailForCursor()
}

// closeProblem returns to the list, leaving the cursor where it was.
func (m Model) closeProblem() (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	m.focus = paneBoard
	m.showEditorial = false
	return m, nil
}

// enterProblem opens the problem screen if it is not already open.
//
// Every verb that acts on ONE problem calls this first — edit, run, submit, editorial.
// They all put their answer somewhere only the problem screen shows: the result panel,
// the submission queue, the statement pane. Run from the list without this and the
// keypress is silent, which reads as broken rather than as "wrong screen".
func (m *Model) enterProblem() {
	if m.mode == modeSolve || m.currentSlug() == "" {
		return
	}
	m.mode = modeSolve
	m.focus = paneDetail
}

// cycleFocus moves between panes, over only the panes the current screen actually has.
//
// On the list there is one pane, so tab does nothing rather than moving focus somewhere
// invisible and making the next keypress behave unexpectedly.
func (m Model) cycleFocus(delta int) (tea.Model, tea.Cmd) {
	if m.mode != modeSolve {
		return m, nil
	}
	// Two panes on this screen: the statement and the working column.
	if m.focus == paneDetail {
		m.focus = paneQueue
	} else {
		m.focus = paneDetail
	}
	return m, nil
}
