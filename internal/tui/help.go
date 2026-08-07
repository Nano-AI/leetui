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
			{"z Z", "reveal tags / hints — hidden, they spoil the approach"},
		}},
		{"solve", [][2]string{
			{"f", "create the solution file, then open it yourself"},
			{"e", "edit the solution in your editor"},
			{"r", "run it locally against the examples"},
			{"s", "submit it to the judge"},
			{"l", "choose the language"},
			{"E", "choose the editor"},
			{"", "saving re-runs the tests; leetui run -w for a pane of its own"},
		}},
		{"premium", [][2]string{
			{"c", "browse company lists, then a timeframe"},
			{"d", "read the official editorial"},
			{"t T", "start-stop the timer, reset it"},
		}},
		{"do", [][2]string{
			{"S", "sync problems — press again to pause"},
			{"v", "the repository — accepted commits itself"},
			{":", "command line — :set default_lang go"},
			{"V", "settings"},
			{"", "pushing lives in there, and asks first"},
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

	body := strings.Join(blocks, "")
	if fold {
		body = helpColumns(blocks, colWidth)
	}
	return head + m.windowHelp(body, foot) + foot
}

// windowHelp scrolls the bindings when they do not fit.
//
// Folding to two columns buys one doubling and then stops, and this screen grows every
// time a feature does. Dropping groups to fit is the one thing it must not do — the whole
// job of this screen is showing every binding — so what does not fit scrolls, and the
// footer says so.
func (m Model) windowHelp(body, foot string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")

	// Everything already spent: the heading, the footer, and a line of slack for the
	// shell prompt that appears under the alternate screen on exit.
	spent := 2 + strings.Count(foot, "\n") + 2
	room := maxInt(m.height-spent, 4)
	if len(lines) <= room {
		return body
	}

	start := clamp(m.helpScroll, 0, maxScroll(len(lines), room-1))
	end := minInt(start+room-1, len(lines))

	shown := strings.Join(lines[start:end], "\n")
	return shown + "\n " + theme.Meta.Render(fmt.Sprintf(
		"%d more — j k to scroll", len(lines)-end+start))
}

func renderHelpGroup(g helpGroup, ruleWidth int) string {
	var b strings.Builder
	b.WriteString("\n " + theme.Utility.Render(theme.UtilityText(g.name)) + " " +
		theme.Rule.Render(strings.Repeat(theme.Chars().DashRule, maxInt(ruleWidth-len(g.name)-3, 1))) + "\n")
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
