package tui

import (
	"github.com/Nano-AI/leetui/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Cursor
// ---------------------------------------------------------------------------

func (m Model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	return m.moveCursorTo(m.cursor + delta)
}

// scrollBy moves by a page or half-page. In the detail pane it scrolls the statement
// instead of the list, so ctrl-d does the expected thing wherever focus happens to be.
func (m Model) scrollBy(delta int) (tea.Model, tea.Cmd) {
	if m.focus == paneDetail {
		m.detailScroll = maxInt(m.detailScroll+delta, 0)
		return m, nil
	}
	return m.moveCursorTo(m.cursor + delta)
}

func (m Model) moveCursorTo(i int) (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	i = clamp(i, 0, len(m.rows)-1)
	if i == m.cursor {
		return m, nil
	}
	m.cursor = i
	m.clampScroll()
	m.detailScroll = 0
	return m, m.loadDetailForCursor()
}

// clampScroll keeps the cursor inside the visible window, scrolling by the minimum
// needed so the list does not jump under the user.
func (m *Model) clampScroll() {
	visible := m.visibleRows()
	if visible <= 0 {
		return
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	m.scroll = clamp(m.scroll, 0, maxInt(len(m.rows)-visible, 0))
}

func (m Model) currentSlug() string {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return ""
	}
	return m.rows[m.cursor].Slug
}

func (m Model) filterActive() bool {
	f := m.filter
	return f.Text != "" || len(f.Difficulty) > 0 || f.Status != "" ||
		len(f.Tags) > 0 || len(f.Companies) > 0 || f.PaidOnly != nil
}

// toggleDifficulty maps 1/2/3 to Easy/Medium/Hard and 0 to "clear".
func (m Model) toggleDifficulty(n int) (tea.Model, tea.Cmd) {
	var want string
	switch n {
	case 0:
		m.filter = store.Filter{}
		m.cursor, m.scroll = 0, 0
		return m, m.loadRows()
	case 1:
		want = "Easy"
	case 2:
		want = "Medium"
	case 3:
		want = "Hard"
	default:
		return m, nil
	}

	next := make([]string, 0, 3)
	found := false
	for _, d := range m.filter.Difficulty {
		if d == want {
			found = true
			continue
		}
		next = append(next, d)
	}
	if !found {
		next = append(next, want)
	}
	m.filter.Difficulty = next
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}
