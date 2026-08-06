package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/runner"
)

// Pickers.
//
// Both are selection lists rather than free text. For languages, only what LeetCode
// offers for THIS problem is valid, and the snippet table already names them. For
// editors, only what is installed can be launched. Typing a name and finding out later
// that it was wrong is a worse experience than not being able to type it.
//
// Each row carries the one fact that decides the choice: whether a language runs here
// or on the judge, and whether an editor keeps you in the terminal.

type pickKind int

const (
	pickNone pickKind = iota
	pickLang
	pickEditor
)

// pickerRow is one entry, reduced to what the list needs to draw.
type pickerRow struct {
	label string
	// note is the right-hand annotation: the reason to choose this row or not.
	note string
	// dim marks a row that works but is not the fast path.
	dim bool
}

// pickerRows builds the current picker's entries.
func (m Model) pickerRows() []pickerRow {
	switch m.picking {
	case pickEditor:
		rows := make([]pickerRow, 0, len(m.editors))
		for _, e := range m.editors {
			note, dim := "in terminal", false
			if e.GUI {
				note, dim = "opens a window", true
			}
			rows = append(rows, pickerRow{label: e.Name, note: note, dim: dim})
		}
		return rows

	default:
		langs := m.pickerLangs()
		rows := make([]pickerRow, 0, len(langs))
		for _, l := range langs {
			note, dim := "runs here", false
			switch {
			case m.engine.Supports(l):
			case l.Local:
				note, dim = "needs a toolchain", true
				if local, ok := m.engine.(*runner.Local); ok {
					if bin := local.MissingToolchain(l); bin != "" {
						note = "needs " + bin
					}
				}
			default:
				note, dim = "judge only", true
			}
			rows = append(rows, pickerRow{label: l.Display, note: note, dim: dim})
		}
		return rows
	}
}

// pickerLangs is the language list for the selected problem, or everything when nothing
// is selected. LeetCode does not offer every language on every problem.
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
	n := len(m.pickerRows())

	switch msg.String() {
	case "esc", "q", "ctrl+c":
		m.picking = pickNone
		return m, nil

	case "j", "down":
		if m.pickIdx < n-1 {
			m.pickIdx++
		}
		return m, nil

	case "k", "up":
		if m.pickIdx > 0 {
			m.pickIdx--
		}
		return m, nil

	case "enter":
		return m.choose()
	}
	return m, nil
}

// choose applies the highlighted entry.
func (m Model) choose() (tea.Model, tea.Cmd) {
	kind := m.picking
	m.picking = pickNone

	switch kind {
	case pickEditor:
		if m.pickIdx >= len(m.editors) {
			return m, nil
		}
		chosen := m.editors[m.pickIdx]
		m.cfg.Editor = chosen.Command

		// Persist it: an editor chosen once should not have to be chosen again.
		if err := m.cfg.Save(); err != nil {
			return m, status("Chose "+chosen.Name+", but could not save it: "+err.Error(), true)
		}
		return m, status("Editor set to "+chosen.Name+". Press e to open the solution.", false)

	default:
		langs := m.pickerLangs()
		if m.pickIdx >= len(langs) {
			return m, nil
		}
		m.lang = langs[m.pickIdx]
		// A result from the previous language says nothing about this one.
		m.runResult, m.runSlug = nil, ""
		return m, status("Language set to "+m.lang.Display+".", false)
	}
}
