package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/Nano-AI/leetui/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAuditRowsRejectOutOfOrderResults(t *testing.T) {
	m := newTestModel(t, true)
	m.booted = true
	old := m.loadRows()()
	next, cmd := m.toggleDifficulty(3)
	m = next.(Model)
	fresh := cmd()
	next, _ = m.Update(fresh)
	m = next.(Model)
	if len(m.rows) != 1 || m.currentSlug() != "trapping-rain-water" {
		t.Fatal("new filter result was not applied")
	}
	next, _ = m.Update(old)
	m = next.(Model)
	if len(m.rows) != 1 || m.currentSlug() != "trapping-rain-water" {
		t.Fatal("late unfiltered query replaced the difficulty results")
	}
	staleErr := old.(rowsMsg)
	staleErr.err = errors.New("old query failed")
	_, cmd = m.Update(staleErr)
	if cmd != nil {
		t.Fatal("stale query error produced a status message")
	}
}

// Run batched commands backwards to exercise the legal schedule in which a read
// finishes before a sibling write. Follow row messages only; never run detail fetches.
func auditApplyCommands(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for i := len(msg) - 1; i >= 0; i-- {
			m = auditApplyCommands(m, msg[i])
		}
	case rowsMsg:
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestAuditMarkRefreshFollowsWrite(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		m := newTestModel(t, true)
		m.booted = true
		if err := m.store.SetMark(context.Background(), "lru-cache", store.MarkDown, ""); err != nil {
			t.Fatal(err)
		}
		m.marks = map[string]store.Mark{"lru-cache": store.MarkDown}
		if filtered {
			m.filter.Mark = store.MarkDown
		}
		m = auditApplyCommands(m, m.loadRows())
		for i, row := range m.rows {
			if row.Slug == "lru-cache" {
				m.cursor = i
			}
		}
		next, cmd := m.toggleMark(store.MarkUp)
		m = auditApplyCommands(next.(Model), cmd)
		if filtered && len(m.rows) != 0 {
			t.Error("mark read raced its write: excluded row still present")
		}
		if !filtered && m.rows[0].Slug != "lru-cache" {
			t.Error("important row did not float after marking")
		}
	}
}

func TestAuditSameFilterRefreshKeepsSelection(t *testing.T) {
	m := newTestModel(t, true)
	m.booted = true
	next, _ := m.Update(m.loadRows()())
	m = next.(Model)
	m.cursor = 2 // LRU Cache
	old := m.loadRows()()
	if err := m.store.SetMark(context.Background(), "lru-cache", store.MarkUp, ""); err != nil {
		t.Fatal(err)
	}
	next, _ = m.Update(m.loadRows()())
	m = next.(Model)
	if m.currentSlug() != "lru-cache" || m.cursor != 0 {
		t.Fatalf("refresh moved selection: %q at %d", m.currentSlug(), m.cursor)
	}
	next, _ = m.Update(old)
	m = next.(Model)
	if m.rows[0].Slug != "lru-cache" {
		t.Fatal("older same-filter snapshot undid the refresh")
	}
}

func TestAuditResizeRejectsOldRender(t *testing.T) {
	m := newTestModel(t, false)
	m.rows = []store.Row{{Slug: "p"}}
	d := &store.Detail{Content: "<p>body</p>"}
	d.Slug = "p"
	m.width = 80
	old := m.renderDetail(d)()
	m.width = 120
	fresh := m.renderDetail(d)()
	next, _ := m.Update(fresh)
	m = next.(Model)
	m.detailMD = "new render"
	next, _ = m.Update(old)
	if next.(Model).detailMD != "new render" {
		t.Fatal("old-width render overwrote current statement")
	}
}
