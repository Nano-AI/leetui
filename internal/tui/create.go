package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
)

// Creating the solution file, and nothing else.
//
// `e` lays the file out AND launches an editor. This does only the first half, because
// under D-016 the editor is already open somewhere else: what is actually wanted is for
// the file to exist, scaffolded and ready, with its path on screen to open by hand.
//
// It is deliberately not "touch". The file arrives with its imports, its package clause,
// its driver include, and the marked region (D-014), plus the folder's README and test
// cases — everything that makes it openable rather than merely present.

// startCreate asks which language, then creates that file.
//
// The picker is always shown rather than assumed. Which language you want is the one
// decision creating a file involves, and its default is already whatever you used last —
// so the fast path is two keys, and the choice is never taken away.
func (m Model) startCreate() (tea.Model, tea.Cmd) {
	if _, _, bail := m.ready(); bail != nil {
		return m, bail
	}
	m.picking, m.pickIdx = pickCreate, m.langIndex()
	return m, nil
}

// langIndex is where the picker's cursor starts: on the current language, which New seeds
// from the remembered one.
func (m Model) langIndex() int {
	for i, l := range m.pickerLangs() {
		if l.Slug == m.lang.Slug {
			return i
		}
	}
	return 0
}

// createFile lays out the problem folder for a language and reports where it went.
func (m Model) createFile(lang runner.Lang) (tea.Model, tea.Cmd) {
	d, _, bail := m.ready()
	if bail != nil {
		return m, bail
	}

	m.lang = lang
	// A result from the previous language says nothing about this one.
	m.runResult, m.runSlug = nil, ""

	// Whether the file already existed decides the wording. "Created" over a file you
	// have been working in for an hour would be alarming, and it is the same call.
	_, existed := os.Stat(m.solutionPath())

	out, err := solve.Prepare(m.cfg.Workspace, d, lang)
	if err != nil {
		return m, status("Could not create the file: "+err.Error(), true)
	}

	title := "created " + lang.Display
	if existed == nil {
		title = lang.Display + " ready"
	}

	// Adopt the file's timestamp so the watcher treats the next SAVE as the first change,
	// rather than firing a run on the scaffolding that was just written.
	m.noteSolutionChanged()

	return m, tea.Batch(
		m.rememberLang(lang),
		notify(title, prettyPath(out.Solution)),
	)
}

// rememberLang persists the choice so the next problem starts where this one left off.
//
// Written to config rather than held in memory: "the one I used last" is only useful if
// it survives a restart, and a language is a habit rather than a per-session decision.
func (m Model) rememberLang(lang runner.Lang) tea.Cmd {
	if m.cfg.LastLang == lang.Slug {
		return nil
	}
	cfg := m.cfg
	cfg.LastLang = lang.Slug
	return func() tea.Msg {
		if err := cfg.Save(); err != nil {
			// Not worth interrupting over: the file was created, which is what was asked.
			return nil
		}
		return nil
	}
}

// prettyPath shortens a path for display, keeping the end that identifies the file.
func prettyPath(path string) string {
	return fitPath(path, 60)
}
