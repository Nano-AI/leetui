package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// The reference.
//
// `?` answers "what does this key do" — it is a keymap, read while your hands are on the
// keys. This answers a different question: "what can this thing DO", including the parts
// that are not keys at all. The command line, the subcommands, and the settings are all
// real surfaces with no key to press, so a keymap cannot list them and they were
// effectively invisible.
//
// Which is the failure this exists to fix: every feature added in the last session was
// reachable and none of it was findable.

// docSection is one titled block of the reference.
type docSection struct {
	name string
	// note is one line under the heading, for the rule that governs the whole section.
	note string
	rows [][2]string
}

func docSections() []docSection {
	return []docSection{
		{
			name: "the command line",
			note: "press : — tab completes, esc backs out",
			rows: [][2]string{
				{":set <key> <value>", "change a setting, written to config.toml"},
				{":set <key>", "a boolean with no value toggles"},
				{":set keys.run x", "rebind an action; a duplicate is refused"},
				{":settings", "the same options as a list you can browse"},
				{":sync", "fetch the problem set"},
				{":git", "the repository view"},
				{":help", "the keymap"},
			},
		},
		{
			name: "reading a problem",
			note: "tags and hints are hidden — both give the approach away",
			rows: [][2]string{
				{"z", "reveal this problem's tags"},
				{"Z", "reveal its hints, one at a time"},
				{"d", "the official editorial (premium)"},
				{"o", "open it on leetcode.com"},
				{"1-9", "open a numbered figure in your browser"},
				{"", "searching by tag still works either way"},
			},
		},
		{
			name: "from your shell",
			note: "the same core the app uses — see docs/AGENTS.md",
			rows: [][2]string{
				{"leetui pull <p>", "lay out the folder: statement, solution, cases"},
				{"leetui run [p]", "run locally; 0 passed, 1 wrong, 2 could not run"},
				{"leetui run --watch", "stay open and re-run on save, in its own pane"},
				{"leetui submit [p]", "send it to the judge — real and public"},
				{"leetui path <p>", "print the folder, for cd and scripts"},
				{"leetui image <p> [n]", "draw a figure, on kitty / iTerm2 / WezTerm"},
				{"leetui todo add <p>", "queue a problem from anywhere"},
				{"leetui todo --json", "the queue, for agents"},
				{"leetui doctor", "what this machine can do, and how to fix what it cannot"},
			},
		},
		{
			name: "the working pane",
			note: "leetui is the LeetCode side of the desk; the code is yours",
			rows: [][2]string{
				{"f", "create the solution file and show its path"},
				{"e", "open it — a new tmux pane where possible"},
				{"r", "run it against the examples"},
				{"s", "submit; an accepted verdict commits itself"},
				{"l  E", "choose the language / the editor"},
				{"", "saving re-runs the tests on its own"},
			},
		},
		{
			name: "languages",
			note: "everything else edits and submits normally, and runs on the judge",
			rows: [][2]string{
				{"local", "Python · Go · C++ · JavaScript · TypeScript"},
				{"", "leetui doctor says which are installed here"},
			},
		},
	}
}

func (m Model) openDocs() (tea.Model, tea.Cmd) {
	m.mode, m.docsScroll = modeDocs, 0
	return m, nil
}

func (m Model) handleDocsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "ctrl+c":
		m.mode = modeBoard
		return m, nil
	case "j", "down":
		m.docsScroll++
		return m, nil
	case "k", "up":
		m.docsScroll = maxInt(m.docsScroll-1, 0)
		return m, nil
	case "g":
		m.docsScroll = 0
		return m, nil
	case "?":
		m.mode, m.helpScroll = modeHelp, 0
		return m, nil
	}
	return m, nil
}

// viewDocs renders the reference.
//
// Folds to two columns before it scrolls. The whole complaint this answers is that the
// surface was hard to find, and a reference whose most-asked-about section starts below
// the fold has reproduced the problem one level in.
func (m Model) viewDocs() string {
	sections := docSections()

	// The widest entry decides the column so descriptions line up and can be scanned
	// down rather than hunted for.
	keyW := 0
	for _, s := range sections {
		for _, r := range s.rows {
			keyW = maxInt(keyW, len(r[0]))
		}
	}

	rows := 0
	for _, s := range sections {
		rows += len(s.rows) + 3
	}

	const chrome = 6
	// minCol is the narrowest a column can be before an entry and its description stop
	// fitting side by side.
	const minCol = 48

	colWidth := maxInt(minInt(m.width-2, 80), 24)
	fold := rows+chrome > m.height && m.width >= 2*minCol+4
	if fold {
		colWidth = minInt((m.width-4)/2, 60)
		keyW = minInt(keyW, colWidth/2)
	}
	keyW = minInt(keyW, maxInt(colWidth/2, 10))

	blocks := make([]string, len(sections))
	for i, s := range sections {
		blocks[i] = renderDocSection(s, keyW, colWidth)
	}

	head := "\n " + lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("reference")) + "\n"
	foot := "\n " +
		theme.Label.Render("?") + theme.Body.Render(" keys   ") +
		theme.Label.Render("j k") + theme.Body.Render(" scroll   ") +
		theme.Label.Render("esc") + theme.Body.Render(" back") + "\n"

	body := strings.Join(blocks, "")
	if fold {
		body = helpColumns(blocks, colWidth)
	}
	return head + m.windowDocs(body, foot) + foot
}

// renderDocSection is one titled block.
func renderDocSection(s docSection, keyW, ruleWidth int) string {
	var b strings.Builder
	b.WriteString("\n " + theme.Utility.Render(theme.UtilityText(s.name)) + " " +
		theme.Rule.Render(strings.Repeat(theme.Chars().DashRule,
			maxInt(ruleWidth-len(s.name)-3, 1))) + "\n")
	if s.note != "" {
		b.WriteString("   " + theme.Meta.Render(truncate(s.note, maxInt(ruleWidth-4, 10))) + "\n")
	}
	for _, r := range s.rows {
		b.WriteString(fmt.Sprintf("   %s  %s\n",
			theme.Label.Render(padRight(truncate(r[0], keyW), keyW)),
			theme.Body.Render(truncate(r[1], maxInt(ruleWidth-keyW-6, 8)))))
	}
	return b.String()
}

// windowDocs scrolls whatever still does not fit after folding.
func (m Model) windowDocs(body, foot string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")

	spent := 2 + strings.Count(foot, "\n") + 2
	room := maxInt(m.height-spent, 4)
	if len(lines) <= room {
		return body
	}

	start := clamp(m.docsScroll, 0, maxScroll(len(lines), room-1))
	end := minInt(start+room-1, len(lines))
	return strings.Join(lines[start:end], "\n") + "\n " + theme.Meta.Render(fmt.Sprintf(
		"%d more — j k to scroll", len(lines)-end+start))
}
