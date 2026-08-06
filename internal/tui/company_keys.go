package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// handleCompanyKey drives the company browser.
//
// Typing filters. It does NOT need a search key first: a list of 984 companies has one
// obvious thing to do with a keystroke, and making the user press "/" before typing
// "goog" would be ceremony. Navigation therefore uses arrows and ctrl+n/ctrl+p rather
// than j/k, which have to stay available as letters.
func (m Model) handleCompanyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.visibleCompanies()

	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBoard
		m.companyFilter.Blur()
		return m, nil

	case "down", "ctrl+n":
		if m.companyIdx < len(rows)-1 {
			m.companyIdx++
		}
		return m, nil

	case "up", "ctrl+p":
		if m.companyIdx > 0 {
			m.companyIdx--
		}
		return m, nil

	case "enter":
		if m.companyIdx >= len(rows) {
			return m, nil
		}
		return m.chooseCompany(rows[m.companyIdx])
	}

	// Anything else edits the filter. Re-anchor the cursor: leaving it on row 40 after
	// the list shrinks to three rows points at nothing.
	before := m.companyFilter.Value()
	var cmd tea.Cmd
	m.companyFilter, cmd = m.companyFilter.Update(msg)
	if m.companyFilter.Value() != before {
		m.companyIdx = 0
	}
	return m, cmd
}

// chooseCompany moves on to the timeframe question, first reading how much of each
// window is already stored so the picker can say which are instant.
func (m Model) chooseCompany(c store.Company) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	m.companyFilter.Blur()
	m.packChoice = c
	m.picking, m.pickIdx = pickTimeframe, 0
	return m, m.loadPackCounts(c.Slug)
}

// loadPackCounts reads the stored size of each of a company's five timeframes.
func (m Model) loadPackCounts(company string) tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		counts := make(map[leetcode.Timeframe]int, len(leetcode.Timeframes()))
		for _, tf := range leetcode.Timeframes() {
			n, err := st.PackCount(ctx, company, string(tf))
			if err != nil {
				return packCountsMsg{company: company, err: err}
			}
			counts[tf] = n
		}
		return packCountsMsg{company: company, counts: counts}
	}
}
