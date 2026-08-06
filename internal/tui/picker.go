package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// The language picker.
//
// A selection list rather than free text: only the languages LeetCode offers for THIS
// problem are valid, and the snippet table already names them. Typing "pythn" and
// getting an error later is a worse experience than not being able to type it.
//
// Each entry says whether it runs locally, because that is the difference between a
// fast offline loop and a network round-trip per run (D-004).

// pickerLangs is the list for the selected problem, or the full set when nothing is
// selected yet.
func (m Model) pickerLangs() []runner.Lang {
	if m.detail == nil || len(m.detail.Snippets) == 0 {
		return runner.Langs()
	}
	slugs := make([]string, 0, len(m.detail.Snippets))
	for slug := range m.detail.Snippets {
		slugs = append(slugs, slug)
	}
	return runner.Available(slugs)
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	langs := m.pickerLangs()

	switch msg.String() {
	case "esc", "q", "ctrl+c":
		m.picking = false
		return m, nil

	case "j", "down":
		if m.pickIdx < len(langs)-1 {
			m.pickIdx++
		}
		return m, nil

	case "k", "up":
		if m.pickIdx > 0 {
			m.pickIdx--
		}
		return m, nil

	case "enter":
		if m.pickIdx < len(langs) {
			m.lang = langs[m.pickIdx]
			// A result from the previous language says nothing about this one.
			m.runResult, m.runSlug = nil, ""
		}
		m.picking = false
		return m, status("Language set to "+m.lang.Display+".", false)
	}
	return m, nil
}

// viewPicker draws the picker centred over the board.
func (m Model) viewPicker() string {
	langs := m.pickerLangs()
	inner := minInt(maxInt(m.width-20, 34), 46)

	f := components.Frame{
		Title:   "language",
		Right:   m.lang.Display,
		Width:   inner + 2,
		Height:  minInt(len(langs), 12) + 4,
		Focused: true,
	}

	var b strings.Builder
	b.WriteString("\n")

	start := 0
	if m.pickIdx >= 12 {
		start = m.pickIdx - 11
	}
	for i := start; i < len(langs) && i-start < 12; i++ {
		l := langs[i]

		marker := "  "
		name := theme.Body.Render(l.Display)
		if i == m.pickIdx {
			marker = theme.Label.Render("▌ ")
			name = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).Render(l.Display)
		}

		// "runs here" is the fact worth surfacing: it decides whether a run costs
		// milliseconds or a network round-trip.
		note := theme.Meta.Render("judge only")
		if m.engine.Supports(l) {
			note = theme.Label.Render("runs here")
		} else if l.Local {
			if local, ok := m.engine.(*runner.Local); ok {
				if bin := local.MissingToolchain(l); bin != "" {
					note = theme.Meta.Render("needs " + bin)
				}
			}
		}

		pad := inner - lipgloss.Width(marker) - lipgloss.Width(name) - lipgloss.Width(note) - 2
		if pad < 1 {
			pad = 1
		}
		b.WriteString(fmt.Sprintf(" %s%s%s%s\n", marker, name, strings.Repeat(" ", pad), note))
	}

	hint := " " + theme.Meta.Render("enter  choose") +
		theme.Rule.Render("  ┊  ") + theme.Meta.Render("esc  cancel")

	return lipgloss.JoinVertical(lipgloss.Left, "", f.Render(b.String()), hint)
}
