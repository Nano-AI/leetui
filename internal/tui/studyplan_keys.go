package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handlePlanKey drives the study plan browser.
//
// Typing filters, for the same reason it does in the company browser: there is one
// obvious thing to do with a keystroke in a list of plans, and requiring "/" first would
// be ceremony. Navigation is arrows and ctrl+n/ctrl+p so the letters stay typeable.
func (m Model) handlePlanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.visiblePlans()

	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBoard
		m.planFilter.Blur()
		return m, nil

	case "down", "ctrl+n":
		if m.planIdx < len(rows)-1 {
			m.planIdx++
		}
		return m, nil

	case "up", "ctrl+p":
		if m.planIdx > 0 {
			m.planIdx--
		}
		return m, nil

	case "enter":
		if m.planIdx >= len(rows) {
			return m, nil
		}
		// One step, unlike a company pack: there is no timeframe to ask about, so enter
		// applies the plan rather than opening a second picker.
		return m.applyPlan(rows[m.planIdx])
	}

	// Anything else edits the filter. Re-anchor the cursor: leaving it on row 12 after the
	// list shrinks to two rows points at nothing.
	before := m.planFilter.Value()
	var cmd tea.Cmd
	m.planFilter, cmd = m.planFilter.Update(msg)
	if m.planFilter.Value() != before {
		m.planIdx = 0
	}
	return m, cmd
}
