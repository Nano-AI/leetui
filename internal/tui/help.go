package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewHelp() string {
	groups := []struct {
		name string
		rows [][2]string
	}{
		{"move", [][2]string{
			{"j k ↑ ↓", "up and down"},
			{"g G", "first and last"},
			{"tab", "next pane"},
			{"enter", "open in the detail pane"},
		}},
		{"find", [][2]string{
			{"/", "search titles, tags, and statements"},
			{"1 2 3", "toggle easy, medium, hard"},
			{"u", "cycle all → unsolved → solved"},
			{"0 esc", "clear every filter"},
		}},
		{"do", [][2]string{
			{"S", "sync problems — press again to pause"},
			{"a", "sign in with session cookies"},
			{"o", "open the problem on leetcode.com"},
			{"1-9", "open image N, in the problem pane"},
			{"t T", "start-stop the timer, reset it"},
		}},
	}

	var b strings.Builder
	b.WriteString("\n " + lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("keys")) + "\n")

	for _, g := range groups {
		b.WriteString("\n " + theme.Utility.Render(theme.UtilityText(g.name)) + " " +
			theme.Rule.Render(strings.Repeat("╌", maxInt(minInt(m.width, 60)-len(g.name)-3, 1))) + "\n")
		for _, r := range g.rows {
			b.WriteString(fmt.Sprintf("   %s  %s\n",
				theme.Label.Width(10).Render(r[0]),
				theme.Body.Render(r[1])))
		}
	}

	b.WriteString("\n " + theme.Meta.Render("Arrows and page keys always work. Every binding is remappable in") + "\n")
	b.WriteString(" " + theme.Meta.Render(m.cfg.Path()) + "\n")
	b.WriteString("\n " + theme.Label.Render("? or esc") + theme.Body.Render("  back") + "\n")
	return b.String()
}
