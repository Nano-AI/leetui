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
func (m Model) boardHeight() int {
	if m.width < narrowMin {
		return m.bodyHeight()
	}
	return m.bodyHeight() / 2
}

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

	if m.picking {
		return m.viewPicker()
	}

	switch m.mode {
	case modeSetup:
		return m.viewSetup()
	case modeHelp:
		return m.viewHelp()
	case modeAuth:
		return m.viewAuth()
	}

	var sections []string
	sections = append(sections, m.viewRail())
	if m.searching {
		sections = append(sections, m.viewSearchPanel())
	}
	sections = append(sections, m.viewBody())
	sections = append(sections, m.viewStatus())
	sections = append(sections, m.viewHints())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) viewBody() string {
	h := m.bodyHeight()

	switch {
	case m.width < narrowMin:
		if m.focus == paneDetail {
			return m.viewDetail(m.width, h)
		}
		return m.viewBoard(m.width, h)

	case m.width < wideMin:
		boardH := m.boardHeight()
		return lipgloss.JoinVertical(lipgloss.Left,
			m.viewBoard(m.width, boardH),
			m.viewDetail(m.width, h-boardH),
		)

	default:
		boardH := m.boardHeight()
		lowerH := h - boardH
		detailW := m.width * 3 / 5
		queueW := m.width - detailW
		return lipgloss.JoinVertical(lipgloss.Left,
			m.viewBoard(m.width, boardH),
			lipgloss.JoinHorizontal(lipgloss.Top,
				m.viewDetail(detailW, lowerH),
				m.viewQueue(queueW, lowerH),
			),
		)
	}
}
