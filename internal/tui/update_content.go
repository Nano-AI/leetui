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
		if msg.slug != m.currentSlug() || msg.seq != m.editorialSeq {
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
		// Editorials are where nearly all the figures are.
		return m, m.fetchPaneImages()

	case todoMsg:
		if msg.err == nil {
			m.todo = msg.slugs
		}
		return m, nil

	case marksMsg:
		// A failed read leaves whatever is in memory alone. The marks already on screen
		// are correct until proven otherwise, and blanking the column on a transient
		// error would look like the verdicts had been lost.
		if msg.err == nil {
			m.marks = msg.marks
		}
		return m, nil

	case companiesMsg:
		if msg.err != nil {
			return m, status("Could not read the company list: "+msg.err.Error(), true)
		}
		m.companies = msg.companies
		if n := len(m.visibleCompanies()); m.companyIdx >= n {
			m.companyIdx = maxInt(n-1, 0)
		}
		return m, nil

	case packCountsMsg:
		// Stale: another company was picked while this was in flight.
		if msg.err != nil || msg.company != m.packChoice.Slug {
			return m, nil
		}
		m.packCounts = msg.counts
		return m, nil

	case plansMsg:
		if msg.err != nil {
			return m, status("Could not read the study plans: "+msg.err.Error(), true)
		}
		m.plans = msg.plans
		if n := len(m.visiblePlans()); m.planIdx >= n {
			m.planIdx = maxInt(n-1, 0)
		}
		return m, nil

	case planGroupsMsg:
		// Stale: another plan was chosen while this was in flight. Labelling rows with a
		// previous plan's chapters would be worse than labelling none.
		if msg.err != nil || msg.plan != m.plan.Slug {
			return m, nil
		}
		m.plan.Groups = msg.groups
		return m, nil

	case contestsMsg:
		if msg.err != nil {
			return m, status("Could not read the contest schedule: "+msg.err.Error(), true)
		}
		m.contests = msg.contests
		if n := len(m.visibleContests()); m.contestIdx >= n {
			m.contestIdx = maxInt(n-1, 0)
		}
		return m, nil

	case contestRegistrationMsg:
		// Silent unless the answer is both known and bad. An error means signed out or
		// unreachable, and a warning nobody can act on is a warning that gets ignored.
		// Stale: the board has moved to another contest since this was asked.
		if msg.err != nil || msg.registered || msg.contest != m.contest.Slug {
			return m, nil
		}
		return m, status("NOT REGISTERED for "+m.contest.Title+
			" — submissions will score nothing. Register at leetcode.com/contest/"+
			m.contest.Slug+"/", true)

	}
	return m, nil
}
