package tui

import (
	"testing"
	"time"
)

// TestWatchFirstSightingDoesNotRun: selecting a problem solved last week must not
// immediately fire a run just because the file exists.
func TestWatchFirstSightingDoesNotRun(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.WatchSolution = true
	m.detail = detailWithSnippets("python3")
	m.watched = map[string]time.Time{}

	model, _ := m.handleWatchTick()
	got := model.(Model)
	if got.running {
		t.Error("a first sighting started a run")
	}
}

// TestWatchIgnoresUnchangedFile: polling must not re-run on every tick.
func TestWatchIgnoresUnchangedFile(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.WatchSolution = true
	m.detail = detailWithSnippets("python3")

	stamp := time.Now()
	m.watched = map[string]time.Time{m.detail.Slug + "/python3": stamp}

	model, _ := m.handleWatchTick()
	if model.(Model).running {
		t.Error("an unchanged file started a run")
	}
}

// TestWatchDisabledDoesNothing: the poll must stay cheap and inert when switched off.
func TestWatchDisabledDoesNothing(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.WatchSolution = false
	m.detail = detailWithSnippets("python3")

	model, cmd := m.handleWatchTick()
	if model.(Model).running {
		t.Error("watching ran while disabled")
	}
	if cmd == nil {
		t.Error("the poll stopped rescheduling itself")
	}
}

// TestWatchKeepsPollingWhileRunning: a tick during a run must not stack a second one,
// but must keep the loop alive.
func TestWatchKeepsPollingWhileRunning(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.WatchSolution = true
	m.detail = detailWithSnippets("python3")
	m.running = true

	_, cmd := m.handleWatchTick()
	if cmd == nil {
		t.Error("polling stopped while a run was in flight")
	}
}
