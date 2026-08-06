package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Editorials (D-006)
// ---------------------------------------------------------------------------
//
// The editorial shares the detail pane rather than opening a fourth one. They are two
// readings of the same problem and are never wanted side by side in a terminal's width;
// showEditorial says which is on screen and the same scroll and number keys serve both.

// toggleEditorial switches the detail pane between statement and editorial.
func (m Model) toggleEditorial() (tea.Model, tea.Cmd) {
	slug := m.currentSlug()
	if slug == "" {
		return m, nil
	}

	if m.showEditorial {
		m.showEditorial = false
		m.detailScroll = 0
		return m, nil
	}

	m.showEditorial = true
	m.detailScroll = 0
	m.focus = paneDetail

	// Already loaded for this problem: no request, no loading state.
	if m.editorial != nil && m.editorial.Slug == slug {
		return m, nil
	}

	m.editorial, m.editorialMD, m.editorialImages = nil, "", nil
	m.editorialLoading = true
	return m, m.fetchEditorial(slug)
}

// fetchEditorial pulls an official solution, from the cache when it is there.
//
// Editorials are large and most are never read, so they are fetched one at a time on
// request rather than synced in bulk — the same bargain as problem statements (D-009).
func (m Model) fetchEditorial(slug string) tea.Cmd {
	sy := m.sync
	width := m.detailWidth()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		e, err := sy.Editorial(ctx, slug, false)
		if e == nil {
			return editorialMsg{slug: slug, err: err}
		}
		if e.Content == "" {
			// Gated, or an editorial that exists with nothing readable in it. The pane
			// renders the lock from the row itself.
			return editorialMsg{slug: slug, editorial: e, err: err}
		}
		md, images, rerr := render.Editorial(e.Content, width)
		if rerr != nil {
			return editorialMsg{slug: slug, editorial: e, err: rerr}
		}
		return editorialMsg{slug: slug, editorial: e, markdown: md, images: images, err: err}
	}
}

// paneImages is the asset list the number keys open — whichever reading is on screen.
//
// Without this, pressing 2 on an editorial would open the second figure of the
// statement, which is a different picture entirely.
func (m Model) paneImages() []render.Image {
	if m.showEditorial {
		return m.editorialImages
	}
	return m.detailImages
}

// dropEditorial clears editorial state when the cursor moves to another problem.
//
// The pane STAYS in editorial mode: someone reading editorials down a list wants the
// next one, not a silent switch back to statements.
func (m *Model) dropEditorial(slug string) {
	if m.editorial != nil && m.editorial.Slug == slug {
		return
	}
	m.editorial, m.editorialMD, m.editorialImages = nil, "", nil
	m.editorialLoading = false
}

// editorialFor returns the cached editorial if it belongs to the row on screen.
func (m Model) editorialFor(row store.Row) (*store.Editorial, bool) {
	if m.editorial != nil && m.editorial.Slug == row.Slug {
		return m.editorial, true
	}
	return nil, false
}
