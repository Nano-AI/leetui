package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// viewPicker draws whichever picker is open.
func (m Model) viewPicker() string {
	rows := m.pickerRows()
	inner := minInt(maxInt(m.width-20, 36), 52)

	title, right := "language", m.lang.Display
	if m.picking == pickEditor {
		title, right = "editor", "installed here"
	}

	const visible = 12
	f := components.Frame{
		Title:   title,
		Right:   right,
		Width:   inner + 2,
		Height:  minInt(len(rows), visible) + 4,
		Focused: true,
	}

	start := 0
	if m.pickIdx >= visible {
		start = m.pickIdx - visible + 1
	}

	var b strings.Builder
	b.WriteString("\n")
	for i := start; i < len(rows) && i-start < visible; i++ {
		b.WriteString(m.pickerLine(rows[i], i == m.pickIdx, inner))
	}

	hint := " " + theme.Meta.Render("enter  choose") +
		theme.Rule.Render("  ┊  ") + theme.Meta.Render("esc  cancel")

	return lipgloss.JoinVertical(lipgloss.Left, "", f.Render(b.String()), hint)
}

func (m Model) pickerLine(r pickerRow, selected bool, inner int) string {
	marker := "  "
	label := theme.Body.Render(r.label)
	if selected {
		marker = theme.Label.Render("▌ ")
		label = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).Render(r.label)
	}

	note := theme.Label.Render(r.note)
	if r.dim {
		note = theme.Meta.Render(r.note)
	}

	pad := inner - lipgloss.Width(marker) - lipgloss.Width(label) - lipgloss.Width(note) - 2
	if pad < 1 {
		pad = 1
	}
	return fmt.Sprintf(" %s%s%s%s\n", marker, label, strings.Repeat(" ", pad), note)
}
