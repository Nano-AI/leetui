package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// Toasts.
//
// The status line at the bottom is for things that comment on what you just did. A toast
// is for something that HAPPENED and that you now need — a file that exists where it did
// not a moment ago, and whose path you are about to type into another window.
//
// It floats over the top right because that is where nothing else lives: the rail's right
// end carries the sync note, the timer, and the account, all of which are glanceable
// rather than read. A toast under them is out of the way of the list and the statement,
// which are not.

// toastLife is how long a toast stays up.
//
// Long enough to read a path and retype it, short enough that it is gone before it
// becomes furniture. It is dismissible either way — any keypress clears it.
const toastLife = 8 * time.Second

// toast is a floating notice.
type toast struct {
	title string
	body  string
	// id distinguishes generations, so an expiry for a toast that has already been
	// replaced does not clear the new one.
	id int
}

type (
	// toastMsg raises a toast.
	toastMsg struct{ title, body string }
	// clearToastMsg expires one.
	clearToastMsg struct{ id int }
)

// notify raises a toast with a heading and a line of detail.
func notify(title, body string) tea.Cmd {
	return func() tea.Msg { return toastMsg{title: title, body: body} }
}

// showToast stores a toast and returns the command that expires it.
func (m *Model) showToast(title, body string) tea.Cmd {
	m.toastID++
	m.toast = &toast{title: title, body: body, id: m.toastID}
	id := m.toastID
	return tea.Tick(toastLife, func(time.Time) tea.Msg { return clearToastMsg{id: id} })
}

// viewToast renders the floating notice, or "" when there is none.
func (m Model) viewToast() string {
	if m.toast == nil {
		return ""
	}

	// Sized to the body, since the body is the part that has to be read exactly — a
	// wrapped path is a path you cannot retype.
	inner := minInt(maxInt(lipgloss.Width(m.toast.body)+2, lipgloss.Width(m.toast.title)+2), m.width-8)
	if inner < 12 {
		return ""
	}

	f := components.Frame{
		Title:   m.toast.title,
		Width:   inner + 2,
		Height:  3,
		Focused: true,
	}
	return f.Render(" " + theme.Body.Render(truncate(m.toast.body, inner-2)))
}

// withToast composites the toast over a finished frame.
//
// Applied last, over the completed view, so a toast can never disturb the layout
// underneath it — the frame is already solved by the time this runs.
func (m Model) withToast(view string) string {
	box := m.viewToast()
	if box == "" {
		return view
	}
	// Flush to the right edge, and starting on the body's first row.
	//
	// An inset would leave the pane's own corner stranded beside the toast — a lone "╮"
	// floating next to a box reads as a rendering fault. Flush, the toast's border simply
	// replaces the pane's for the rows it covers.
	return components.OverlayTopRight(view, box, railHeight, 0)
}
