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
			note: "press : · tab completes",
			rows: [][2]string{
				{":set <key> <value>", "change a setting, saved to disk"},
				{":set <key>", "no value toggles a boolean"},
				{":set keys.run x", "rebind a key"},
				{":settings", "the options as a browsable list"},
				{":sync", "fetch the problem set"},
				{":git", "the repository view"},
				{":help", "the keymap"},
			},
		},
		{
			name: "reading a problem",
			note: "hidden — they spoil the approach",
			rows: [][2]string{
				{"z", "reveal tags for this problem"},
				{"Z", "reveal its hints"},
				{"d", "the official editorial"},
				{"o", "open it on leetcode.com"},
				{"1-9", "open figure N in your browser"},
				{":set ui.inline_images true", "draw figures in the pane"},
				{"", "search by tag still works"},
			},
		},
		{
			name: "from your shell",
			note: "same core · see docs/AGENTS.md",
			rows: [][2]string{
				{"leetui pull <p>", "lay out the problem folder"},
				{"leetui run [p]", "run locally · exit 0 1 2"},
				{"leetui run --watch", "re-run on save, in its own pane"},
				{"leetui submit [p]", "to the judge — real and public"},
				{"leetui path <p>", "print the folder, for scripts"},
				{"leetui image <p> [n]", "draw figure N (kitty, iTerm2)"},
				{"leetui todo add <p>", "queue a problem from anywhere"},
				{"leetui todo --json", "the queue, as JSON"},
				{"leetui doctor", "what works here, and what to fix"},
				{"leetui --debug", "trace requests, credentials redacted"},
			},
		},
		{
			name: "the working pane",
			note: "the code is yours; this is not",
			rows: [][2]string{
				{"f", "create the file, show its path"},
				{"e", "open it, in a tmux pane"},
				{"r", "run against the examples"},
				{"s", "accepted commits itself"},
				{"l  E", "choose language / editor"},
				{"", "saving re-runs the tests"},
			},
		},
		{
			name: "languages",
			note: "the rest runs on the judge",
			rows: [][2]string{
				{"local", "Python Go C++ JS TS"},
				{"", "doctor says which you have"},
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
