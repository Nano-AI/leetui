package tui

import (
	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

// handleKey dispatches through the config-driven action table (D-013). Arrows and
// PgUp/PgDn are an always-on fallback and are deliberately not remappable.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Modal states consume keys first.
	switch {
	case m.mode == modeSetup:
		// Setup is not a prison: quitting works, and esc drops into the board to browse
		// whatever has already landed while the sync keeps running.
		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "enter":
			m.mode = modeBoard
			return m, m.loadRows()
		}
		return m, nil

	case m.picking != pickNone:
		return m.handlePickerKey(msg)

	case m.mode == modeCompany:
		return m.handleCompanyKey(msg)
	case m.paletteOpen:
		return m.handlePaletteKey(msg)
	case m.mode == modeSettings:
		return m.handleSettingsKey(msg)
	case m.mode == modeDocs:
		return m.handleDocsKey(msg)
	case m.mode == modeGit:
		return m.handleGitKey(msg)
	case m.mode == modeAuth:
		return m.handleAuthKey(msg)
	case m.searching:
		return m.handleSearchKey(msg)
	}

	// Fallback navigation, always available regardless of the keymap.
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "down":
		return m.moveCursor(1)
	case "up":
		return m.moveCursor(-1)
	case "pgdown":
		return m.moveCursor(m.visibleRows())
	case "pgup":
		return m.moveCursor(-m.visibleRows())
	case "home":
		return m.moveCursorTo(0)
	case "end":
		return m.moveCursorTo(len(m.rows) - 1)
	case "esc":
		if m.mode == modeHelp {
			m.mode = modeBoard
			return m, nil
		}
	}

	// Number keys are context-sensitive: on the list they toggle difficulty filters, on a
	// problem they open the numbered markers. The SCREEN decides, which is plainer than
	// the old rule where focus did — you can see which screen you are on.
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		if m.mode == modeSolve {
			return m.openImage(int(key[0] - '0'))
		}
		return m.toggleDifficulty(int(key[0] - '0'))
	}

	switch m.keys[key] {
	case "quit":
		return m, tea.Quit

	case "help":
		if m.mode == modeHelp {
			m.mode = modeBoard
		} else {
			m.mode, m.helpScroll = modeHelp, 0
		}
		return m, nil

	case "focus_next":
		return m.cycleFocus(1)
	case "focus_prev":
		return m.cycleFocus(-1)

	case "down":
		if m.mode == modeHelp {
			m.helpScroll++
			return m, nil
		}
		if m.focus == paneDetail {
			m.detailScroll = minInt(m.detailScroll+1, m.detailMaxScroll())
			return m, nil
		}
		return m.moveCursor(1)
	case "up":
		if m.mode == modeHelp {
			m.helpScroll = maxInt(m.helpScroll-1, 0)
			return m, nil
		}
		if m.focus == paneDetail {
			m.detailScroll = maxInt(m.detailScroll-1, 0)
			return m, nil
		}
		return m.moveCursor(-1)
	case "half_down":
		return m.scrollBy(m.visibleRows() / 2)
	case "half_up":
		return m.scrollBy(-m.visibleRows() / 2)
	case "page_down":
		return m.scrollBy(m.visibleRows())
	case "page_up":
		return m.scrollBy(-m.visibleRows())

	case "top":
		if m.focus == paneDetail {
			m.detailScroll = 0
			return m, nil
		}
		return m.moveCursorTo(0)
	case "bottom":
		// Mirrors "top", which has always scrolled the statement when the statement is
		// focused. G moving the board cursor instead was the odd one out.
		if m.focus == paneDetail {
			m.detailScroll = m.detailMaxScroll()
			return m, nil
		}
		return m.moveCursorTo(len(m.rows) - 1)

	case "search":
		m.searching = true
		m.search.SetValue(m.filter.Text)
		m.search.Focus()
		return m, textinput.Blink

	case "edit":
		return m.startEdit()
	case "create":
		return m.startCreate()
	case "todo":
		return m.toggleTodo()
	case "todo_only":
		return m.toggleTodoFilter()
	case "run":
		return m.startRun()
	case "submit":
		return m.startSubmit()
	case "lang":
		if len(m.rows) > 0 {
			m.picking, m.pickIdx = pickLang, 0
		}
		return m, nil

	case "editor":
		return m.openEditorPicker()

	case "open":
		return m.openProblem()

	case "editorial":
		return m.toggleEditorial()
	case "companies":
		return m.openCompanies()

	case "sync":
		return m.startSync()

	case "palette":
		return m.openPalette()

	case "settings":
		return m.openSettings()

	case "docs":
		return m.openDocs()

	case "tags":
		return m.toggleSpoiler("ui.show_tags")

	case "hints":
		return m.toggleSpoiler("ui.show_hints")

	case "git":
		return m.openGit()

	case "auth":
		m.mode = modeAuth
		m.authErr = ""
		m.importing = ""
		m.authInput.SetValue("")
		m.authInput.Focus()
		// Detection is pure filesystem stats — no keychain, so no prompt until the
		// user actually picks a browser.
		m.browsers = auth.DetectBrowsers()
		return m, textinput.Blink

	case "open_web":
		return m.openInBrowser()

	case "timer":
		m.timerRunning = !m.timerRunning
		return m, nil
	case "timer_reset":
		m.elapsed, m.timerRunning = 0, false
		return m, nil

	case "back":
		// esc peels one layer at a time, innermost first. Dropping straight to an
		// unfiltered board would throw away a pack the user spent two choices building.
		if m.showEditorial {
			return m.toggleEditorial()
		}
		if m.mode == modeSolve {
			return m.closeProblem()
		}
		if m.filterActive() {
			return m.clearPack()
		}
		return m, nil
	}

	// Unbound single-letter conveniences that do not deserve a remappable action.
	switch key {
	case "u":
		return m.cycleStatus()
	case "p":
		return m.cyclePremium()
	}

	return m, nil
}
