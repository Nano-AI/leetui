package tui

import (
	"github.com/Nano-AI/leetui/internal/auth"
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

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Digits pick a detected browser. They are checked before the text field so the
	// picker works without a modifier; typing a digit into a pasted cookie is not a
	// thing anyone does, since cookies are pasted whole.
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' && m.authInput.Value() == "" {
		if n := int(key[0] - '1'); n < len(m.browsers) {
			return m.importFromBrowser(m.browsers[n])
		}
	}

	switch key {
	case "esc", "ctrl+c":
		m.mode = modeBoard
		m.authInput.Blur()
		m.authInput.SetValue("")
		return m, nil

	case "enter":
		creds, err := auth.Parse(m.authInput.Value())
		if err != nil {
			m.authErr = err.Error()
			return m, nil
		}
		if err := auth.Store(creds); err != nil {
			m.authErr = err.Error()
			return m, nil
		}
		m.client.SetCredentials(creds)
		m.mode = modeBoard
		m.authInput.Blur()
		// Clear the pasted secret from memory as soon as it is stored.
		m.authInput.SetValue("")
		return m, tea.Batch(m.loadAccount(), status("Signed in. Press S to sync.", false))
	}

	var cmd tea.Cmd
	m.authInput, cmd = m.authInput.Update(msg)
	return m, cmd
}
