package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// viewContests is the contest browser: a filter field over a framed schedule.
//
// Ordered newest first, which puts the next contest and the last one at the top. That is
// the opposite of the plan browser's "started first" and it is the right rule here: a
// contest's relevance is its date, and the two you might want are both at that end.
func (m Model) viewContests() string {
	rows := m.visibleContests()
	width := maxInt(m.width, 4)

	filter := components.Frame{
		Title:   "contest",
		Right:   fmt.Sprintf("%d of %d", len(rows), len(m.contests)),
		Width:   width,
		Height:  3,
		Focused: true,
	}

	visible := maxInt(minInt(m.height-12, 16), 4)
	list := components.Frame{
		Title:  "contests",
		Width:  width,
		Height: visible + 2,
	}

	start := 0
	if m.contestIdx >= visible {
		start = m.contestIdx - visible + 1
	}

	var b strings.Builder
	switch {
	case len(m.contests) == 0 && m.syncing:
		b.WriteString(" " + theme.Meta.Render("Loading the contest schedule…") + "\n")
	case len(rows) == 0:
		b.WriteString(" " + theme.Meta.Render("No contest matches that.") + "\n")
	}
	now := time.Now()
	for i := start; i < len(rows) && i-start < visible; i++ {
		b.WriteString(m.contestLine(rows[i], i == m.contestIdx, list.InnerWidth(), now))
	}

	hint := " " + theme.Meta.Render("type to narrow") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("↑↓ move") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("enter  pick") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("esc  back")

	return lipgloss.JoinVertical(lipgloss.Left, "",
		filter.Render(" "+m.contestFilter.View()),
		list.Render(b.String()),
		lipgloss.NewStyle().Width(width).Render(hint))
}

// contestLine is one schedule row: the contest, when it runs, and where it stands.
//
// A LIVE contest is the only row that gets amber. It is the one you can still score in,
// and on a screen of twenty grey rows the eye should land on it without reading.
func (m Model) contestLine(c store.Contest, selected bool, inner int, now time.Time) string {
	marker, nameStyle := "  ", theme.Body
	if selected {
		marker = theme.Label.Render(theme.Chars().Cursor + " ")
		nameStyle = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true)
	}

	phase := c.PhaseAt(now)
	var note string
	switch {
	case phase == leetcode.PhaseLive:
		note = theme.Label.Render(formatDuration(c.Remaining(now)) + " left")
	case phase == leetcode.PhaseUpcoming:
		note = theme.Meta.Render("in " + formatDuration(c.Remaining(now)))
	case c.Solved > 0:
		note = theme.Label.Render(fmt.Sprintf("%d/%d", c.Solved, c.Stored))
	case c.Stored > 0:
		note = theme.Meta.Render(fmt.Sprintf("%d problems", c.Stored))
	default:
		note = theme.Meta.Render("not pulled")
	}

	name := nameStyle.Render(ansi.Truncate(c.Title, maxInt(inner-lipgloss.Width(note)-5, 0), "…"))
	// The date fills whatever the title and the standing leave behind, and is the first
	// thing cut when narrow: a contest's name already carries its week.
	when := theme.Meta.Render("  " + time.Unix(c.StartTime, 0).Local().Format("Mon 02 Jan 15:04"))
	pad := inner - lipgloss.Width(marker) - lipgloss.Width(name) -
		lipgloss.Width(note) - lipgloss.Width(when) - 2
	if pad < 1 {
		when = ""
		pad = maxInt(inner-lipgloss.Width(marker)-lipgloss.Width(name)-lipgloss.Width(note)-2, 1)
	}
	return fmt.Sprintf(" %s%s%s%s%s\n", marker, name, when, strings.Repeat(" ", pad), note)
}
