package tui

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// launchEditor starts the editor where planEdit decided it should go.
//
// Arguments go as a slice in every branch, never a shell string — the paths are built
// from a LeetCode slug.
func (m Model) launchEditor(p editPlan) (tea.Model, tea.Cmd) {
	switch p.route {
	case routePane:
		// The multiplexer's control binary creates the pane and exits, so this returns
		// immediately even though the editor keeps running. leetui never leaves the
		// screen: the statement stays up and the watcher keeps re-running on save.
		if err := p.splitter.Command(p.dir, p.argv).Run(); err != nil {
			// The pane did not open — Kitty without remote control is the usual reason.
			// Fall back to taking over the terminal rather than leaving the user with a
			// keypress that did nothing.
			return m, tea.Batch(
				status(fallbackNote(p.splitter.Name, err), true),
				takeover(p),
			)
		}
		// Adopt the file's current timestamp so the first save is what triggers a run,
		// not the scaffolding write that just happened.
		m.noteSolutionChanged()
		return m, status(p.note(), false)

	case routeDetached:
		cmd := exec.Command(p.argv[0], p.argv[1:]...)
		if err := cmd.Start(); err != nil {
			return m, status("Could not start "+p.editor.Name+": "+err.Error(), true)
		}
		// Released rather than waited on: leetui stays interactive while the window is
		// open, and the process is the user's to close.
		_ = cmd.Process.Release()
		m.noteSolutionChanged()
		return m, status(p.note(), false)

	default:
		return m, takeover(p)
	}
}

// takeover hands the terminal to the editor and takes it back cleanly.
//
// Bubbletea restores the alt screen and mouse mode around the child process; leaving
// either on while a full-screen editor runs corrupts both.
func takeover(p editPlan) tea.Cmd {
	cmd := exec.Command(p.argv[0], p.argv[1:]...)
	cmd.Dir = p.dir
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editDoneMsg{err: err}
	})
}

// noteSolutionChanged records the solution file's current timestamp and reports whether
// it moved since the last look.
//
// The watcher and the after-edit run read the same map, which is what stops them both
// firing for one save. Whoever notices the change first claims it.
func (m *Model) noteSolutionChanged() bool {
	if m.detail == nil {
		return false
	}
	ws, err := workspace.New(m.cfg.Workspace)
	if err != nil {
		return false
	}
	mod, err := ws.ModTime(m.detail.NumericID, m.detail.Slug, m.lang.Filename())
	if err != nil || mod.IsZero() {
		return false
	}

	if m.watched == nil {
		m.watched = map[string]time.Time{}
	}
	key := m.detail.Slug + "/" + m.lang.Slug
	previous, seen := m.watched[key]
	m.watched[key] = mod
	return seen && mod.After(previous)
}
