package tui

import (
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// editorialBody is the detail pane's content when the editorial is on screen.
//
// Every branch here says what would be there and what to do about it. A gate that only
// showed nothing would be indistinguishable from a problem with no editorial at all,
// which is the failure D-006 exists to prevent.
func (m Model) editorialBody(row store.Row) string {
	if m.editorialMD != "" && m.editorial != nil && m.editorial.Slug == row.Slug {
		return m.editorialMD
	}

	e, loaded := m.editorialFor(row)

	switch {
	case loaded && !e.CanSee:
		return m.editorialLocked(e)

	case m.editorialLoading:
		return theme.Meta.Render("Loading the editorial…")

	case loaded:
		return theme.Meta.Render("LeetCode has no editorial for this problem.") + "\n\n" +
			theme.Meta.Render("d  back to the statement")

	default:
		return theme.Meta.Render("Press d again to load the editorial.")
	}
}

// editorialLocked names what is behind the gate rather than reporting a failure.
func (m Model) editorialLocked(e *store.Editorial) string {
	title := e.Title
	if title == "" {
		title = "This editorial"
	}

	body := theme.Label.Render(title+" is Premium.") + "\n\n" +
		theme.Body.Render("The official write-up, its complexity analysis, and the\n"+
			"reference implementations load with a Premium session.")

	if e.HasVideo {
		body += "\n\n" + theme.Body.Render("There is a video walkthrough too.")
	}

	return body + "\n\n" +
		theme.Meta.Render("a  sign in with a premium account") + "\n" +
		theme.Meta.Render("o  open it in your browser") + "\n" +
		theme.Meta.Render("d  back to the statement")
}
