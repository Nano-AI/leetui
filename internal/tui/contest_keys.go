package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleContestKey drives the contest browser.
//
// The same shape as the plan browser — typing filters, arrows and ctrl+n/ctrl+p move,
// enter picks — because a third list with a fourth set of habits would be one too many.
func (m Model) handleContestKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.visibleContests()

	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBoard
		m.contestFilter.Blur()
		return m, nil

	case "down", "ctrl+n":
		if m.contestIdx < len(rows)-1 {
			m.contestIdx++
		}
		return m, nil

	case "up", "ctrl+p":
		if m.contestIdx > 0 {
			m.contestIdx--
		}
		return m, nil

	case "enter":
		if m.contestIdx >= len(rows) {
			return m, nil
		}
		return m.applyContest(rows[m.contestIdx])
	}

	// Anything else edits the filter. Re-anchor the cursor: leaving it on row 12 after the
	// list shrinks to two rows points at nothing.
	before := m.contestFilter.Value()
	var cmd tea.Cmd
	m.contestFilter, cmd = m.contestFilter.Update(msg)
	if m.contestFilter.Value() != before {
		m.contestIdx = 0
	}
	return m, cmd
}
