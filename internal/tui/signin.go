package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// viewAuth is the sign-in panel.
//
// Two routes, cheapest first. Importing from a browser is one keypress; pasting is the
// portable fallback that works everywhere and needs no permissions.
//
// The earlier version dumped a paragraph of instructions that ran past the frame and
// broke the border. Every line here is short enough to fit, and the prose that survived
// is directions rather than explanation.
func (m Model) viewAuth() string {
	inner := minInt(maxInt(m.width-8, 44), 60)
	f := components.Frame{
		Title:   "sign in",
		Width:   inner + 2,
		Height:  m.authPanelHeight(),
		Focused: true,
	}

	var b strings.Builder
	b.WriteString("\n")

	// --- Route 1: import -----------------------------------------------------
	b.WriteString(" " + theme.Utility.Render(theme.UtilityText("import from browser")) + "\n")
	switch {
	case m.importing != "":
		b.WriteString("  " + theme.Label.Render("Reading "+m.importing+"…") + "\n")
		b.WriteString("  " + theme.Meta.Render("Allow the keychain prompt if it appears.") + "\n")
	case len(m.browsers) == 0:
		b.WriteString("  " + theme.Meta.Render("No supported browser found.") + "\n")
	default:
		for i, br := range m.browsers {
			if i >= 9 {
				break
			}
			b.WriteString("  " + theme.Label.Render(fmt.Sprintf("%d", i+1)) + "  " +
				theme.Body.Render(br.Label()) + "\n")
		}
		b.WriteString("  " + theme.Meta.Render("Reads only your leetcode.com cookies.") + "\n")
	}

	b.WriteString(" " + theme.Rule.Render(strings.Repeat(theme.Chars().DashRule, maxInt(inner-2, 1))) + "\n")

	// --- Route 2: paste ------------------------------------------------------
	b.WriteString(" " + theme.Utility.Render(theme.UtilityText("or paste cookies")) + "\n")
	b.WriteString(" " + theme.Label.Render(theme.Chars().Cursor + " ") + m.authInput.View() + "\n")
	for _, line := range auth.PasteSteps() {
		b.WriteString("  " + theme.Meta.Render(truncate(line, inner-3)) + "\n")
	}

	if m.authErr != "" {
		b.WriteString("\n " + theme.Label.Render(truncate(m.authErr, inner-2)) + "\n")
	}

	hint := " " + theme.Meta.Render("enter  save") +
		theme.Rule.Render(sep2()) + theme.Meta.Render("esc  cancel")

	return lipgloss.JoinVertical(lipgloss.Left, "", f.Render(b.String()), hint,
		" "+theme.Meta.Render("Cookies go to your OS keychain, never to disk."))
}

// authPanelHeight sizes the panel to its content so the frame never clips a line.
func (m Model) authPanelHeight() int {
	rows := 1 // leading blank
	rows++    // "import from browser"
	switch {
	case m.importing != "":
		rows += 2
	case len(m.browsers) == 0:
		rows++
	default:
		rows += minInt(len(m.browsers), 9) + 1
	}
	rows++ // divider
	rows++ // "or paste cookies"
	rows++ // input
	rows += len(auth.PasteSteps())
	if m.authErr != "" {
		rows += 2
	}
	return rows + 2 // bezels
}
