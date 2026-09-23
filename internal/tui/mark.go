package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Importance marks (D-032)
// ---------------------------------------------------------------------------
//
// `+` says a problem was worth doing, `-` says it was not, and pressing the same key
// again takes the verdict back. Three states, because "no opinion yet" is the honest
// default for a board of four thousand problems.
//
// This is not the todo list. A todo is a queue that empties as you solve; a mark outlives
// the solve, which is what makes a second pass through a study plan possible: keep the
// ones that taught something, skip the ones that did not.

// toggleMark records a verdict on the selected problem, or clears it.
//
// Two keys rather than one, unlike `m` for todo. Todo is binary and one key can carry
// both directions; a verdict has three states, and a single key cycling
// up → down → clear would make "mark this important" a variable number of presses that
// depends on state you would have to read off the screen first.
func (m Model) toggleMark(want store.Mark) (tea.Model, tea.Cmd) {
	slug := m.currentSlug()
	if slug == "" {
		return m, nil
	}

	title := slug
	if m.cursor < len(m.rows) {
		title = m.rows[m.cursor].Title
	}

	// Pressing the key a row already carries takes the verdict back, matching how `m`
	// toggles a todo off.
	next := want
	if m.marks[slug] == want {
		next = store.MarkNone
	}

	// In memory first so the glyph lands on this frame rather than after a round trip to
	// SQLite — this is pressed while scanning, and it has to feel immediate.
	if m.marks == nil {
		m.marks = map[string]store.Mark{}
	}
	if next == store.MarkNone {
		delete(m.marks, slug)
	} else {
		m.marks[slug] = next
	}

	st := m.store
	// The mark changes every ordering, even on an unfiltered board. Query only
	// after the write completes; sibling commands in a Batch run concurrently.
	reload := m.loadRows()
	write := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var err error
		if next == store.MarkNone {
			err = st.ClearMark(ctx, slug)
		} else {
			err = st.SetMark(ctx, slug, next, "")
		}
		if err != nil {
			return statusMsg{text: "Could not save the mark: " + err.Error(), isError: true}
		}
		return reload()
	}

	var note string
	switch next {
	case store.MarkUp:
		note = title + " is important."
	case store.MarkDown:
		note = title + " is unimportant."
	default:
		note = "Cleared the mark on " + title + "."
	}

	return m, tea.Batch(write, status(note, false))
}

// cycleMarkFilter walks the board through all → important → unimportant → all.
//
// One key rather than two, unlike the marking keys: filtering is a deliberate act you
// perform while looking at the result, so cycling costs nothing and saves a binding.
func (m Model) cycleMarkFilter() (tea.Model, tea.Cmd) {
	switch m.filter.Mark {
	case store.MarkNone:
		m.filter.Mark = store.MarkUp
	case store.MarkUp:
		m.filter.Mark = store.MarkDown
	default:
		m.filter.Mark = store.MarkNone
	}

	// Sort by verdict while a mark filter is on, so the most recently judged sit together
	// at the top. Restore the default order when it comes off.
	if m.filter.Mark != store.MarkNone {
		m.filter.Sort = "mark"
	} else if m.filter.Sort == "mark" {
		m.filter.Sort = m.filteredOrder()
	}
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}

// loadMarks reads every verdict, keyed by slug.
func (m Model) loadMarks() tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		marks, err := st.MarkMap(ctx)
		return marksMsg{marks: marks, err: err}
	}
}
