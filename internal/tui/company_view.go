package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// viewCompanies is the company browser: a filter field over a framed list.
//
// Companies are ordered by list size, not alphabetically. In a 984-row list the ones
// people prepare for are the large ones, and putting Amazon and Google at the top means
// the common case needs no typing at all.
func (m Model) viewCompanies() string {
	rows := m.visibleCompanies()
	width := minInt(maxInt(m.width-8, 44), 68)

	filter := components.Frame{
		Title:   "company",
		Right:   fmt.Sprintf("%d of %d", len(rows), len(m.companies)),
		Width:   width,
		Height:  3,
		Focused: true,
	}

	visible := maxInt(minInt(m.height-12, 16), 4)
	list := components.Frame{
		Title:  "lists",
		Right:  "premium",
		Width:  width,
		Height: visible + 2,
	}

	start := 0
	if m.companyIdx >= visible {
		start = m.companyIdx - visible + 1
	}

	var b strings.Builder
	switch {
	case len(m.companies) == 0 && m.syncing:
		b.WriteString(" " + theme.Meta.Render("Loading the company list…") + "\n")
	case len(rows) == 0:
		b.WriteString(" " + theme.Meta.Render("No company matches that.") + "\n")
	}
	for i := start; i < len(rows) && i-start < visible; i++ {
		b.WriteString(m.companyLine(rows[i], i == m.companyIdx, list.InnerWidth()))
	}

	hint := " " + theme.Meta.Render("type to narrow") +
		theme.Rule.Render("  ┊  ") + theme.Meta.Render("↑↓ move") +
		theme.Rule.Render("  ┊  ") + theme.Meta.Render("enter  pick a timeframe") +
		theme.Rule.Render("  ┊  ") + theme.Meta.Render("esc  back")

	return lipgloss.JoinVertical(lipgloss.Left, "",
		filter.Render(" "+m.companyFilter.View()),
		list.Render(b.String()),
		hint)
}

// companyLine is one registry row: the name, how many problems LeetCode lists, and how
// many are already stored here.
//
// The stored count is the one that changes what pressing enter does — it is the
// difference between an instant filter and a network pull — so it is what the row says
// on the right.
func (m Model) companyLine(c store.Company, selected bool, inner int) string {
	marker, name := "  ", theme.Body.Render(c.Name)
	if selected {
		marker = theme.Label.Render("▌ ")
		name = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).Render(c.Name)
	}

	note := theme.Meta.Render(fmt.Sprintf("%d problems", c.QuestionCount))
	if c.Synced > 0 {
		note = theme.Label.Render(fmt.Sprintf("%d stored", c.Synced)) +
			theme.Meta.Render(fmt.Sprintf(" of %d", c.QuestionCount))
	}

	pad := inner - lipgloss.Width(marker) - lipgloss.Width(name) - lipgloss.Width(note) - 2
	if pad < 1 {
		pad = 1
	}
	return fmt.Sprintf(" %s%s%s%s\n", marker, name, strings.Repeat(" ", pad), note)
}
