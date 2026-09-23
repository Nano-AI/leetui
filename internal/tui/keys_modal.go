package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.search.Blur()
		// Escape abandons the search rather than committing it — the user backed out.
		if m.filter.Text != "" {
			m.filter.Text = ""
			m.cursor, m.scroll = 0, 0
			return m, m.loadRows()
		}
		return m, nil

	case "enter", "down", "ctrl+n":
		// Commit and return focus to the board so the results can be navigated.
		m.searching = false
		m.search.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)

	// Search is a store query, so it can run on every keystroke without touching the
	// network. This is the payoff for D-009.
	if v := m.search.Value(); v != m.filter.Text {
		m.filter.Text = v
		m.cursor, m.scroll = 0, 0
		return m, tea.Batch(cmd, m.loadRows())
	}
	return m, cmd
}
