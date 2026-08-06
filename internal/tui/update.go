package tui

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/syncer"
	"github.com/Nano-AI/leetui/internal/tui/components"
)

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.search.Width = maxInt(m.width-20, 20)
		m.authInput.Width = maxInt(m.width-20, 20)
		m.companyFilter.Width = maxInt(m.width-20, 20)

		// Re-render whichever reading is on screen at the new width. Rendering both
		// would wrap the hidden one to a width it may never be shown at.
		if m.showEditorial && m.editorial != nil {
			return m, m.fetchEditorial(m.editorial.Slug)
		}
		if m.detail != nil {
			return m, m.renderDetail(m.detail)
		}
		return m, nil

	case tickMsg:
		if m.timerRunning {
			m.elapsed += time.Second
		}
		return m, secondTick()

	case rowsMsg:
		if msg.err != nil {
			return m, status("Could not read the local database: "+msg.err.Error(), true)
		}
		m.rows = msg.rows
		if m.cursor >= len(m.rows) {
			m.cursor = maxInt(len(m.rows)-1, 0)
		}
		m.clampScroll()

		// First run: an empty database means there is exactly one useful next action, so
		// take it rather than asking the user to press a key for it. Guarded by booted so
		// a filter that matches nothing never re-enters setup.
		if !m.booted {
			m.booted = true
			if len(m.rows) == 0 && !m.filterActive() {
				m.mode = modeSetup
				return m, m.beginSync()
			}
		}
		return m, tea.Batch(m.loadDetailForCursor(), m.loadTodo())

	case detailMsg:
		// A response for a problem the cursor already left is dropped: statements are
		// fetched over the network and can arrive out of order, and showing the wrong
		// one is worse than showing none.
		if msg.slug != m.currentSlug() || (msg.seq != 0 && msg.seq != m.detailSeq) {
			return m, nil
		}
		m.detailLoading = false
		if msg.err != nil {
			if errors.Is(msg.err, leetcode.ErrPremiumRequired) {
				m.detail, m.detailMD = msg.detail, ""
				return m, nil
			}
			if errors.Is(msg.err, leetcode.ErrSessionExpired) {
				return m, status("Session expired. Press a to re-authenticate.", true)
			}
			return m, status("Could not load the problem: "+msg.err.Error(), true)
		}
		m.detail, m.detailMD, m.detailImages = msg.detail, msg.markdown, msg.images
		m.detailScroll = 0
		return m, nil

	case detailFetchMsg:
		// Stale: the cursor moved on while this was queued.
		if msg.seq != m.detailSeq || msg.slug != m.currentSlug() {
			return m, nil
		}
		if !msg.ready {
			m.detailLoading = true
			next := msg
			next.ready = true
			return m, tea.Tick(detailDebounce, func(time.Time) tea.Msg { return next })
		}
		return m, m.fetchDetail(msg.slug, msg.seq)

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

	case browserImportMsg:
		m.importing = ""
		if msg.err != nil {
			m.authErr = browserImportHint(msg.from, msg.err)
			return m, nil
		}
		if err := auth.Store(msg.creds); err != nil {
			m.authErr = err.Error()
			return m, nil
		}
		m.client.SetCredentials(msg.creds)
		m.mode = modeBoard
		m.authInput.Blur()
		m.authInput.SetValue("")
		return m, tea.Batch(
			m.loadAccount(),
			status("Signed in from "+msg.from.Label()+". Press S to sync.", false),
		)

	case editReadyMsg, editDoneMsg, runFinishedMsg, judgeMsg:
		return m.handleSolveMsg(msg)

	case watchTick:
		return m.handleWatchTick()

	case syncProgressMsg:
		return m.handleSyncProgress(syncer.Progress(msg))

	case accountMsg:
		if msg.err != nil {
			// A failed account check is not worth interrupting the user over; the board
			// works fine signed out.
			return m, nil
		}
		m.username, m.premium = msg.status.Username, msg.status.IsPremium
		return m, nil

	case toastMsg:
		return m, m.showToast(msg.title, msg.body)

	case clearToastMsg:
		if m.toast != nil && m.toast.id == msg.id {
			m.toast = nil
		}
		return m, nil

	case statusMsg:
		m.statusID++
		m.status, m.statusErr = msg.text, msg.isError
		id := m.statusID
		return m, tea.Tick(6*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{id: id}
		})

	case clearStatusMsg:
		if msg.id == m.statusID {
			m.status, m.statusErr = "", false
		}
		return m, nil

	case components.FlipTickMsg:
		var cmds []tea.Cmd
		for i := range m.queue {
			var cmd tea.Cmd
			m.queue[i].flap, cmd = m.queue[i].flap.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// A toast covers content, so the next deliberate action clears it rather than
		// making the user wait out the timer.
		m.toast = nil
		return m.handleKey(msg)
	}

	return m, nil
}
