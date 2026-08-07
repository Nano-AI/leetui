package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// Messages carrying content that was fetched or read.
//
// Split out of Update so its switch stays a routing table. Every one of these has the
// same shape of concern: the answer arrived, is it still the answer to the question the
// user is asking NOW? A cursor moves while a request is in flight, and showing a stale
// result is worse than showing none.
func (m Model) handleContentMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorialMsg:
		// A response for a problem the cursor already left is dropped, for the same
		// reason detailMsg drops one: showing the wrong write-up is worse than none.
		if msg.slug != m.currentSlug() {
			return m, nil
		}
		m.editorialLoading = false
		m.editorial, m.editorialMD, m.editorialImages = msg.editorial, msg.markdown, msg.images
		if msg.err != nil && msg.editorial == nil {
			if errors.Is(msg.err, leetcode.ErrNotFound) {
				return m, status("LeetCode has no editorial for this problem.", false)
			}
			return m, status("Could not load the editorial: "+msg.err.Error(), true)
		}
		return m, nil

	case todoMsg:
		if msg.err == nil {
			m.todo = msg.slugs
		}
		return m, nil

	case companiesMsg:
		if msg.err != nil {
			return m, status("Could not read the company list: "+msg.err.Error(), true)
		}
		m.companies = msg.companies
		if m.companyIdx >= len(m.companies) {
			m.companyIdx = maxInt(len(m.companies)-1, 0)
		}
		return m, nil

	case packCountsMsg:
		// Stale: another company was picked while this was in flight.
		if msg.err != nil || msg.company != m.packChoice.Slug {
			return m, nil
		}
		m.packCounts = msg.counts
		return m, nil

	}
	return m, nil
}
