package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The todo list, in the app.
//
// The same list the CLI writes (see cmd_todo.go), which is the point: an agent queues
// problems from a job description, and they are waiting on the board next time it opens.
//
// It is the user's own data, kept in its own table so a re-sync of LeetCode's problem
// list cannot touch it.

// toggleTodo puts the selected problem on the list, or takes it off.
//
// One key for both directions. Marking is something you do while scanning, and having to
// remember which of two keys applies to the row under the cursor would cost more thought
// than the action is worth.
func (m Model) toggleTodo() (tea.Model, tea.Cmd) {
	slug := m.currentSlug()
	if slug == "" {
		return m, nil
	}

	title := slug
	if m.cursor < len(m.rows) {
		title = m.rows[m.cursor].Title
	}
	on := !m.todo[slug]

	// Update in memory first so the mark appears on this frame rather than after a round
	// trip to SQLite — this is a key pressed while scanning, and it has to feel immediate.
	if m.todo == nil {
		m.todo = map[string]bool{}
	}
	m.todo[slug] = on

	st := m.store
	write := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if on {
			_ = st.AddTodo(ctx, slug, "")
		} else {
			_ = st.RemoveTodo(ctx, slug)
		}
		return nil
	}

	note := "Took " + title + " off your list."
	if on {
		note = "Added " + title + " to your list. Press M to see just the list."
	}

	// A filtered list has to re-query: an unmarked row must leave a list-only view.
	if m.filter.TodoOnly {
		return m, tea.Sequence(write, m.loadRows())
	}
	return m, tea.Batch(write, status(note, false))
}

// toggleTodoFilter narrows the board to the list, or restores it.
func (m Model) toggleTodoFilter() (tea.Model, tea.Cmd) {
	m.filter.TodoOnly = !m.filter.TodoOnly
	// Oldest first while the list is showing: it is a queue, and the item added three
	// weeks ago is the one most in danger of being forgotten. Restore the default order
	// when it is not.
	if m.filter.TodoOnly {
		m.filter.Sort = "todo"
	} else if m.filter.Sort == "todo" {
		m.filter.Sort = ""
	}
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}

// loadTodo reads the list of marked slugs.
func (m Model) loadTodo() tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		slugs, err := st.TodoSlugs(ctx)
		return todoMsg{slugs: slugs, err: err}
	}
}
