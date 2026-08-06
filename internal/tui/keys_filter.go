package tui

import tea "github.com/charmbracelet/bubbletea"

// Filter cycles.
//
// Both are one key that steps through every state and wraps, rather than a key per
// state. A board filter is something you flick through while looking at the result, and
// three keys for three states would be three things to remember for one question.

// cycleStatus steps progress: all → unsolved → solved → all.
func (m Model) cycleStatus() (tea.Model, tea.Cmd) {
	switch m.filter.Status {
	case "":
		m.filter.Status = "todo"
	case "todo":
		m.filter.Status = "ac"
	default:
		m.filter.Status = ""
	}
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}

// cyclePremium steps the paid-only filter: all → premium → free → all.
//
// Premium-locked problems are worth isolating in both directions: to work through what a
// subscription buys, and to hide what an account cannot open (D-006).
func (m Model) cyclePremium() (tea.Model, tea.Cmd) {
	yes, no := true, false
	switch {
	case m.filter.PaidOnly == nil:
		m.filter.PaidOnly = &yes
	case *m.filter.PaidOnly:
		m.filter.PaidOnly = &no
	default:
		m.filter.PaidOnly = nil
	}
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}
