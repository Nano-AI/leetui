package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/grootbeat/leetui/internal/store"
	"github.com/grootbeat/leetui/internal/tui/components"
	"github.com/grootbeat/leetui/internal/tui/theme"
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
	f := components.Frame{
		Title:   "problem",
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

	body := m.detailBody(row)
	lines := strings.Split(body, "\n")
	start := clamp(m.detailScroll, 0, maxInt(len(lines)-1, 0))
	room := maxInt(f.InnerHeight()-3, 1)
	end := minInt(start+room, len(lines))
	for _, line := range lines[start:end] {
		b.WriteString(" " + line + "\n")
	}
	return f.Render(b.String())
}

// detailBody picks what to show under the heading.
func (m Model) detailBody(row store.Row) string {
	// Only trust the rendered statement if it belongs to the row on screen.
	if m.detailMD != "" && m.detail != nil && m.detail.Slug == row.Slug {
		return m.detailMD
	}

	switch {
	case row.PaidOnly:
		// A gated pane says what it would contain and how to unlock it — never a raw
		// error, never a silently missing feature (D-006).
		return theme.Label.Render("This problem is Premium.") + "\n\n" +
			theme.Body.Render("Its statement, editorial, and company tags load with a\n"+
				"LeetCode Premium session.") + "\n\n" +
			theme.Meta.Render("a  sign in with a premium account") + "\n" +
			theme.Meta.Render("o  open it in your browser")

	case m.detailLoading:
		return theme.Meta.Render("Loading…")

	default:
		return ""
	}
}
