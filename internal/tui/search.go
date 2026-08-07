package tui

import (
	"fmt"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// viewSearchPanel is the search input in its own bezel.
//
// Search gets a frame rather than a bare line because it is a distinct mode: while it is
// open, typing goes here and not to the board. The frame is the boundary that makes that
// obvious without a modal or a mode indicator to read.
func (m Model) viewSearchPanel() string {
	f := components.Frame{
		Title:   "search",
		Right:   m.matchSummary(),
		Width:   m.width,
		Height:  searchHeight,
		Focused: true,
	}

	// No scope hint here — the field's own placeholder already names what is searched,
	// and saying it twice is the kind of doubling this design keeps cutting.
	return f.Render(" " + theme.Label.Render(theme.Chars().Cursor+" ") + m.search.View())
}

func (m Model) matchSummary() string {
	switch n := len(m.rows); {
	case m.filter.Text == "":
		return fmt.Sprintf("%d problems", n)
	case n == 0:
		return "no matches"
	case n == 1:
		return "1 match"
	default:
		return fmt.Sprintf("%d matches", n)
	}
}
