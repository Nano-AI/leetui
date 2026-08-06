package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// Watching the solution file.
//
// D-012 promises that saving from an editor in another pane re-runs the tests, so you
// can keep leetui and nvim side by side without touching leetui at all.
//
// This polls a modification time rather than taking an fsnotify dependency. One stat
// call every 700ms is free, and it avoids the platform-specific failure modes of file
// watchers — editors that write-then-rename, watches lost on directory replacement,
// descriptor limits on Linux.
const watchInterval = 700 * time.Millisecond

// watchTick asks for the solution file's modification time.
type watchTick struct{}

// watchCmd schedules the next poll.
func watchCmd() tea.Cmd {
	return tea.Tick(watchInterval, func(time.Time) tea.Msg { return watchTick{} })
}

// handleWatchTick re-runs the tests when the solution changed on disk.
//
// The first observation only records the timestamp; it never triggers a run. Otherwise
// selecting a problem you solved last week would immediately run it.
func (m Model) handleWatchTick() (tea.Model, tea.Cmd) {
	if !m.cfg.WatchSolution || m.detail == nil || m.running {
		return m, watchCmd()
	}

	ws, err := workspace.New(m.cfg.Workspace)
	if err != nil {
		return m, watchCmd()
	}

	mod, err := ws.ModTime(m.detail.NumericID, m.detail.Slug, m.lang.Filename())
	if err != nil || mod.IsZero() {
		return m, watchCmd()
	}

	key := m.detail.Slug + "/" + m.lang.Slug
	previous, seen := m.watched[key]
	if m.watched == nil {
		m.watched = map[string]time.Time{}
	}
	m.watched[key] = mod

	// First sighting: remember it, do not act on it.
	if !seen || !mod.After(previous) {
		return m, watchCmd()
	}

	model, runCmd := m.startRun()
	return model, tea.Batch(runCmd, watchCmd())
}
