package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ---------------------------------------------------------------------------
// Detail
// ---------------------------------------------------------------------------

// viewDetail renders the selected problem.
//
// The HEADING comes from the board row, which is always in memory, and the BODY comes
// from the fetched statement. That split is what stops the pane flashing: scrolling
// changes the heading instantly and leaves only the statement area to catch up.
// Rendering the heading from the fetched detail meant every cursor move blanked the
// whole pane and repainted it a moment later.
func (m Model) viewDetail(w, h int) string {
	title := "problem"
	if m.showEditorial {
		title = "editorial"
	}
	f := components.Frame{
		Title:   title,
		Width:   w,
		Height:  h,
		Focused: m.focus == paneDetail,
	}

	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return f.Render(" " + theme.Meta.Render(""))
	}
	row := m.rows[m.cursor]

	// %5.1f keeps the bezel label a fixed width, so the frame's top edge does not
	// twitch between a 42.7% problem and a 100.0% one.
	f.Right = fmt.Sprintf("%5.1f%% accepted", row.AcRate)

	var b strings.Builder
	b.WriteString(" " + lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).
		Render(truncate(fmt.Sprintf("%s. %s", row.FrontendID, row.Title), f.InnerWidth()-8)))
	b.WriteString("  " + difficultyOf(row.Difficulty).Render() + "\n")

	// This line is always drawn, even when empty. Letting it come and go shifted the
	// statement up and down by a row as the cursor moved.
	tags := row.Tags
	if len(row.Companies) > 0 {
		tags = row.Companies
	}
	b.WriteString(" " + theme.Meta.Render(truncate(strings.Join(tags, " ┊ "), f.InnerWidth()-2)) + "\n")
	b.WriteString(" " + theme.Rule.Render(strings.Repeat("╌", maxInt(f.InnerWidth()-2, 1))) + "\n")

	// Hard-wrap before splitting, so one line here is one row on screen.
	//
	// Glamour wraps prose but leaves CODE BLOCKS alone, which is right for code and
	// wrong for a pane: an example like `[4,5,0,-2,-3,1], [5], [5,0], …` is one long
	// line, and an unwrapped long line used to shear the bezel off mid-pane. Wrapping
	// also keeps the scroll honest — a line the user scrolls past is a line they saw.
	//
	// Hardwrap rather than word-wrap: this only ever affects lines Glamour already
	// declined to break, which are code, and breaking code at spaces misaligns it.
	body := ansi.Hardwrap(m.detailBody(row), maxInt(f.InnerWidth()-2, 8), false)
	lines := strings.Split(body, "\n")
	room := maxInt(f.InnerHeight()-3, 1)

	// Stop when the LAST screenful is filled, not when the last line is.
	//
	// Clamping to len(lines)-1 let ctrl-d walk the statement off the top of the pane
	// until one line was left staring at an empty box. There is nothing below the end
	// of a problem, so scrolling there is only ever a mistake to undo.
	start := clamp(m.detailScroll, 0, maxScroll(len(lines), room))
	end := minInt(start+room, len(lines))
	for _, line := range lines[start:end] {
		b.WriteString(" " + line + "\n")
	}
	return f.Render(b.String())
}

// maxScroll is the furthest offset that still fills the pane.
func maxScroll(lines, room int) int { return maxInt(lines-room, 0) }

// detailMaxScroll is how far the statement on screen can scroll.
//
// Computed from the same geometry and the same wrap the view uses, because a clamp that
// disagrees with what is drawn stops in the wrong place — which is the bug it is here to
// prevent, one layer up.
func (m Model) detailMaxScroll() int {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return 0
	}
	w, h := m.detailSize()
	f := components.Frame{Width: w, Height: h}

	body := ansi.Hardwrap(m.detailBody(m.rows[m.cursor]), maxInt(f.InnerWidth()-2, 8), false)
	return maxScroll(len(strings.Split(body, "\n")), maxInt(f.InnerHeight()-3, 1))
}

// detailBody picks what to show under the heading.
func (m Model) detailBody(row store.Row) string {
	if m.showEditorial {
		return m.editorialBody(row)
	}

	// Only trust the rendered statement if it belongs to the row on screen.
	if m.detailMD != "" && m.detail != nil && m.detail.Slug == row.Slug {
		return m.detailMD
	}

	switch {
	// The gate is the ACCOUNT, not the problem. A Premium subscriber opening a paid-only
	// problem is doing something entirely ordinary, and telling them it is locked would
	// be both wrong and a dead end — the statement is on its way.
	case row.PaidOnly && !m.premium:
		// A gated pane says what it would contain and how to unlock it — never a raw
		// error, never a silently missing feature (D-006).
		return theme.Label.Render("This problem is Premium.") + "\n\n" +
			theme.Body.Render("Its statement, editorial, and company tags load with a\n"+
				"LeetCode Premium session.") + "\n\n" +
			theme.Meta.Render("a  sign in with a premium account") + "\n" +
			theme.Meta.Render("o  open it in your browser")

	case m.detailLoading, row.PaidOnly:
		return theme.Meta.Render("Loading…")

	default:
		return ""
	}
}
