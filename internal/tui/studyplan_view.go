package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// viewPlans is the study plan browser: a filter field over a framed list.
//
// Plans you have started come first (store.Plans orders them that way), because a plan is
// a thing you are partway through and "where was I" is the question this screen exists to
// answer.
func (m Model) viewPlans() string {
	rows := m.visiblePlans()
	width := maxInt(m.width, 4)

	filter := components.Frame{
		Title:   "study plan",
		Right:   fmt.Sprintf("%d of %d", len(rows), len(m.plans)),
		Width:   width,
		Height:  3,
		Focused: true,
	}

	visible := maxInt(minInt(m.height-12, 16), 4)
	list := components.Frame{
		Title:  "plans",
		Width:  width,
		Height: visible + 2,
	}

	start := 0
	if m.planIdx >= visible {
		start = m.planIdx - visible + 1
	}

	var b strings.Builder
	switch {
	case len(m.plans) == 0 && m.syncing:
		b.WriteString(" " + theme.Meta.Render("Loading the study plans…") + "\n")
	case len(rows) == 0:
		b.WriteString(" " + theme.Meta.Render("No plan matches that.") + "\n")
	}
	for i := start; i < len(rows) && i-start < visible; i++ {
		b.WriteString(m.planLine(rows[i], i == m.planIdx, list.InnerWidth()))
	}

	hint := " " + theme.Meta.Render("type to narrow") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("↑↓ move") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("enter  pick") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("esc  back")

	return lipgloss.JoinVertical(lipgloss.Left, "",
		filter.Render(" "+m.planFilter.View()),
		list.Render(b.String()),
		lipgloss.NewStyle().Width(width).Render(hint))
}

// planLine is one registry row: the plan, and how far through it you are.
//
// Progress is what the row leads with on the right, because it is the thing that changes
// what you do next. A gated plan says PREMIUM there instead — it has no progress to
// report and naming the gate is more useful than a zero.
func (m Model) planLine(p store.Plan, selected bool, inner int) string {
	marker, nameStyle := "  ", theme.Body
	if selected {
		marker = theme.Label.Render(theme.Chars().Cursor + " ")
		nameStyle = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true)
	}

	var note string
	switch {
	case p.PremiumOnly && !m.premium:
		note = theme.Meta.Render("premium")
	case p.Solved > 0:
		note = theme.Label.Render(fmt.Sprintf("%d/%d", p.Solved, p.QuestionNum)) +
			theme.Meta.Render(fmt.Sprintf("  %d%%", p.Progress()))
	default:
		note = theme.Meta.Render(fmt.Sprintf("%d problems", p.QuestionNum))
	}

	// Reserve the status before shortening the name, so the frame cannot clip the gate.
	name := nameStyle.Render(ansi.Truncate(p.Name, maxInt(inner-lipgloss.Width(note)-5, 0), "…"))
	// The pitch fills whatever the name and the progress leave behind. It is the only
	// thing that distinguishes "LeetCode 75" from "Top 100 Liked" at a glance, so it gets
	// the slack rather than being dropped — but it is the first thing cut when narrow.
	pad := inner - lipgloss.Width(marker) - lipgloss.Width(name) - lipgloss.Width(note) - 2
	hl := ""
	if p.Highlight != "" && pad > 6 {
		text := ansi.Truncate(p.Highlight, pad-4, "…")
		hl = theme.Meta.Render("  " + text)
		pad -= lipgloss.Width(hl)
	}
	if pad < 1 {
		pad = 1
	}
	return fmt.Sprintf(" %s%s%s%s%s\n", marker, name, hl, strings.Repeat(" ", pad), note)
}
