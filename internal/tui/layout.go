package tui

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Layout arithmetic
// ---------------------------------------------------------------------------
//
// Every height and width split is derived here, in one place. app.go asks these
// functions rather than recomputing, so the scroll window and the rendered window can
// never disagree — the bug that makes a list cursor vanish off-screen.

const (
	railHeight   = 3 // bezel, wordmark line, bezel
	hintsHeight  = 1
	statusHeight = 1 // always reserved; see chromeHeight
	searchHeight = 3 // framed input: top bezel, field, bottom bezel
	frameChrome  = 2 // a frame's top and bottom bezel
	headerRows   = 2 // column header + its divider
)

// chromeHeight reserves the status row unconditionally.
//
// Letting it appear and disappear resized the board by one row every time a message
// arrived or expired, so the whole list jumped under the cursor. A blank reserved row
// costs one line and keeps the layout still.
func (m Model) chromeHeight() int {
	h := railHeight + hintsHeight + statusHeight
	if m.searching {
		h += searchHeight
	}
	return h
}

func (m Model) bodyHeight() int {
	return maxInt(m.height-m.chromeHeight(), 6)
}

// boardHeight is the full height of the framed problem list.
//
// The list owns the whole body: browsing is its own screen (D-018), so there is nothing
// below it to leave room for.
func (m Model) boardHeight() int { return m.bodyHeight() }

// visibleRows is how many problems fit: the board minus its bezels and column header.
func (m Model) visibleRows() int {
	return maxInt(m.boardHeight()-frameChrome-headerRows, 1)
}

func (m Model) detailWidth() int {
	if m.width < wideMin {
		return maxInt(m.width-4, 20)
	}
	return maxInt(m.width*3/5-4, 20)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.picking != pickNone {
		return m.viewPicker()
	}

	switch m.mode {
	case modeSetup:
		return m.viewSetup()
	case modeHelp:
		return m.viewHelp()
	case modeAuth:
		return m.viewAuth()
	case modeCompany:
		return m.viewCompanies()
	case modeGit:
		return m.viewGit()
	case modeSettings:
		return m.viewSettings()
	}

	var sections []string
	sections = append(sections, m.viewRail())
	if m.searching {
		sections = append(sections, m.viewSearchPanel())
	}
	sections = append(sections, m.viewBody())
	if p := m.viewPalette(); p != "" {
		sections = append(sections, p)
	}
	sections = append(sections, m.viewStatus())
	sections = append(sections, m.viewHints())

	// The toast goes on last, over a finished frame, so it can never disturb the layout
	// underneath it.
	return m.withToast(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

// viewBody draws the screen for the current mode.
//
// TWO SCREENS, not one crowded one (D-018).
//
// Browsing is a LIST — the whole width, nothing else. leetcode.com/problemset is a table
// and nothing else, for the same reason: while you are looking for a problem, a statement
// you have not chosen to read is in the way, and it costs the list 40% of the width it
// wants for titles and tags.
//
// Picking one opens the solve screen, where the statement earns its space because reading
// it is now the job.
func (m Model) viewBody() string {
	h := m.bodyHeight()
	if m.mode == modeSolve {
		return m.viewSolveBody(h)
	}
	return m.viewBoard(m.width, h)
}

// viewSolveBody is the problem screen: statement, and the working column beside it.
func (m Model) viewSolveBody(h int) string {
	// Narrow: one pane at a time, tab switches. There is no width to split.
	if m.width < wideMin {
		if m.focus == paneQueue {
			return m.viewSidePane(m.width, h)
		}
		return m.viewDetail(m.width, h)
	}

	detailW, _ := m.detailSize()
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.viewDetail(detailW, h),
		m.viewSidePane(m.width-detailW, h),
	)
}

// detailSize is the statement pane's frame size.
//
// Shared with the scroll clamp rather than computed twice: the clamp has to know how much
// of the statement is on screen, and geometry that disagrees with the view is how you get
// a scroll that stops in the wrong place.
func (m Model) detailSize() (w, h int) {
	h = m.bodyHeight()
	if m.width < wideMin {
		return m.width, h
	}
	return m.width * 3 / 5, h
}

// viewSidePane is the working column: what file you are editing, then what happened.
//
// The workbench strip is always on top and always present. It carries the solution's path,
// which is the one thing leetui must say out loud when the editing happens somewhere else
// entirely (see workbench.go).
//
// Below it, a fresh local run displaces the submission queue: the run is what the user
// just asked for and is looking at; the queue is history. A result belonging to another
// problem is not shown at all — see hasResult.
func (m Model) viewSidePane(w, h int) string {
	bench := m.viewWorkbench(w)
	rest := h - workbenchHeight

	// Too short to carry both: the path matters more than the history, but a pane with
	// nothing under it is worse than no strip at all.
	if rest < 3 {
		if m.hasResult() {
			return m.viewResult(w, h)
		}
		return m.viewQueue(w, h)
	}

	body := m.viewQueue(w, rest)
	if m.hasResult() {
		body = m.viewResult(w, rest)
	}
	return lipgloss.JoinVertical(lipgloss.Left, bench, body)
}
