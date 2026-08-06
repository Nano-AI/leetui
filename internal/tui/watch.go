package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	// Shared with the after-edit run, deliberately: both read and advance the same
	// timestamp, so whichever notices a save first claims it and the other sees nothing
	// to do. Two copies of this bookkeeping would run the tests twice for one save.
	if !m.noteSolutionChanged() {
		return m, watchCmd()
	}

	model, runCmd := m.startRun()
	return model, tea.Batch(runCmd, watchCmd())
}
