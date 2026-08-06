package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// First run
// ---------------------------------------------------------------------------

// viewSetup is the first-run screen.
//
// A split-flap board runs a flap cascade when it powers up — every cell cycles before
// settling. That is exactly what a first sync is, so the setup screen IS the boot
// sequence rather than a generic progress dialog bolted on the front.
//
// It starts on its own: an empty board with a "press S" prompt asks the user to perform
// a step the app could have taken itself.
func (m Model) viewSetup() string {
	p := m.syncProgress
	inner := minInt(maxInt(m.width-20, 40), 64)

	f := components.Frame{
		Title:   "first run",
		Width:   inner + 2,
		Height:  13,
		Focused: true,
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(center(lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("leetui")), inner) + "\n\n")
	b.WriteString(center(theme.Body.Render("Building your local problem set."), inner) + "\n")
	b.WriteString(center(theme.Meta.Render("Runs once. After this, search is offline and instant."), inner) + "\n\n")

	b.WriteString(center(progressBar(p.Done, p.Total, minInt(inner-8, 44)), inner) + "\n\n")

	label := "connecting to leetcode.com…"
	if p.Total > 0 {
		label = fmt.Sprintf("%d of %d problems", p.Done, p.Total)
	}
	if p.Note != "" && p.Note != "starting" {
		label = p.Note
	}
	b.WriteString(center(theme.Label.Render(label), inner) + "\n")

	frame := f.Render(b.String())
	hint := center(theme.Meta.Render("esc  skip and browse what has loaded"), inner+2)

	return lipgloss.JoinVertical(lipgloss.Left,
		strings.Repeat("\n", maxInt((m.height-16)/2, 0)),
		indent(frame, maxInt((m.width-inner-2)/2, 0)),
		indent(hint, maxInt((m.width-inner-2)/2, 0)),
	)
}

// progressBar draws the fill. Amber, because it is the system speaking — progress is
// never green (docs/DESIGN.md).
func progressBar(done, total, width int) string {
	if width < 4 {
		return ""
	}
	pct := 0.0
	if total > 0 {
		pct = float64(done) / float64(total)
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(width))

	return lipgloss.NewStyle().Foreground(theme.Amber).Render(strings.Repeat("█", filled)) +
		theme.Rule.Render(strings.Repeat("░", width-filled)) +
		"  " + theme.Label.Render(fmt.Sprintf("%3.0f%%", pct*100))
}

func center(s string, width int) string {
	pad := (width - lipgloss.Width(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func indent(s string, n int) string {
	if n <= 0 {
		return s
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}
