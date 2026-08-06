package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// helpGroup is one titled block of bindings.
type helpGroup struct {
	name string
	rows [][2]string
}

// helpGroups is the whole keymap as the user should read it — grouped by intent, not by
// key. Bindings come from config.DefaultKeymap; this is the prose for them (D-013).
func helpGroups() []helpGroup {
	return []helpGroup{
		{"move", [][2]string{
			{"j k ↑ ↓", "up and down"},
			{"g G", "first and last"},
			{"tab", "next pane"},
			{"enter", "open in the detail pane"},
		}},
		{"list", [][2]string{
			{"m", "mark this problem — add it to your list"},
			{"M", "show just your list, oldest first"},
			{"", "leetui todo add <problem> does it from a script"},
		}},
		{"find", [][2]string{
			{"/", "search titles, tags, and statements"},
			{"1 2 3", "toggle easy, medium, hard"},
			{"u", "cycle all → unsolved → solved"},
			{"p", "cycle all → premium → free"},
			{"0 esc", "clear every filter"},
		}},
		{"solve", [][2]string{
			{"f", "create the solution file, then open it yourself"},
			{"e", "edit the solution in your editor"},
			{"r", "run it locally against the examples"},
			{"s", "submit it to the judge"},
			{"l", "choose the language"},
			{"E", "choose the editor"},
			{"", "inside tmux, e opens a pane beside leetui"},
			{"", "saving re-runs the tests on its own"},
		}},
		{"premium", [][2]string{
			{"c", "browse company lists, then a timeframe"},
			{"d", "read the official editorial"},
			{"t T", "start-stop the timer, reset it"},
		}},
		{"do", [][2]string{
			{"S", "sync problems — press again to pause"},
			{"a", "sign in with session cookies"},
			{"o", "open the problem on leetcode.com"},
			{"1-9", "open marker N, in the problem pane"},
		}},
	}
}

// viewHelp lists every binding.
//
// It folds into two columns when the groups would not fit the terminal's height. The
// alternative — dropping groups to fit — would hide bindings from the one screen whose
// whole job is to show them.
func (m Model) viewHelp() string {
	groups := helpGroups()

	// Lines the groups need, counted before rendering so the column decision can be made
	// once and the blocks built at the width that decision implies.
	rows := len(groups) * 2 // a rule line and a blank line per group
	for _, g := range groups {
		rows += len(g.rows)
	}

	// 7 lines of heading, footer, and a line of slack for the shell's own prompt.
	const chrome = 7
	// minCol is the narrowest a column can be before the longest description wraps.
	const minCol = 46

	colWidth := maxInt(minInt(m.width-2, 60), 20)
	fold := rows+chrome > m.height && m.width >= 2*minCol+4
	if fold {
		colWidth = minInt((m.width-4)/2, 60)
	}

	blocks := make([]string, len(groups))
	for i, g := range groups {
		blocks[i] = renderHelpGroup(g, colWidth)
	}

	head := "\n " + lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("keys")) + "\n"
	foot := "\n " + theme.Meta.Render("Arrows and page keys always work. Every binding is remappable in") +
		"\n " + theme.Meta.Render(m.cfg.Path()) +
		"\n\n " + theme.Label.Render("? or esc") + theme.Body.Render("  back") + "\n"

	if !fold {
		return head + strings.Join(blocks, "") + foot
	}
	return head + helpColumns(blocks, colWidth) + foot
}

func renderHelpGroup(g helpGroup, ruleWidth int) string {
	var b strings.Builder
	b.WriteString("\n " + theme.Utility.Render(theme.UtilityText(g.name)) + " " +
		theme.Rule.Render(strings.Repeat("╌", maxInt(ruleWidth-len(g.name)-3, 1))) + "\n")
	for _, r := range g.rows {
		b.WriteString(fmt.Sprintf("   %s  %s\n",
			theme.Label.Width(10).Render(r[0]),
			theme.Body.Render(r[1])))
	}
	return b.String()
}

// helpColumns splits the groups into two stacks of roughly equal height.
//
// It balances by line count rather than by group count: the groups are different sizes,
// and splitting three-and-two would leave one column noticeably longer.
func helpColumns(blocks []string, colWidth int) string {
	lines := make([]int, len(blocks))
	total := 0
	for i, b := range blocks {
		lines[i] = strings.Count(b, "\n")
		total += lines[i]
	}

	split, running := len(blocks), 0
	for i := range blocks {
		if running+lines[i] > total/2 && i > 0 {
			split = i
			break
		}
		running += lines[i]
	}

	col := lipgloss.NewStyle().Width(colWidth + 2)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		col.Render(strings.Join(blocks[:split], "")),
		col.Render(strings.Join(blocks[split:], "")))
}
