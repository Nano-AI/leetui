package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/syncer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestSetupScreen renders the first-run experience. It is driven directly rather than by
// booting an empty store, because a failing sync (which is what the offline transport
// produces) correctly drops straight through to the board.
func TestSetupScreen(t *testing.T) {
	m := boot(t, false, 120, 34)
	m.mode = modeSetup
	m.syncing = true
	m.syncProgress = syncer.Progress{Phase: syncer.PhaseProblems, Done: 1200, Total: 4013}

	out := m.View()
	t.Logf("\n%s", out)

	for _, want := range []string{"L E E T U I", "1200 of 4013", "30%", "esc"} {
		if !strings.Contains(stripANSI(out), want) {
			t.Errorf("setup screen is missing %q", want)
		}
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 120 {
			t.Errorf("setup line %d overflows: %d cols", i, w)
		}
	}
}

// TestFirstRunStartsSyncItself: an empty database has exactly one useful next action, so
// the app takes it instead of asking the user to press a key for it.
//
// The transition is asserted directly rather than through boot(), because the offline
// transport used in tests fails the sync immediately and the model correctly falls back
// to the board before boot() returns.
func TestFirstRunStartsSyncItself(t *testing.T) {
	m := newTestModel(t, false)
	m.width, m.height = 120, 34

	var model tea.Model = m
	model, cmd := model.Update(rowsMsg{rows: nil})
	got := model.(Model)

	if got.mode != modeSetup {
		t.Errorf("mode = %v on an empty database, want modeSetup", got.mode)
	}
	if !got.syncing {
		t.Error("the first run did not start a sync on its own")
	}
	if cmd == nil {
		t.Error("no command issued to pump sync progress")
	}
	if !got.booted {
		t.Error("boot flag was not set")
	}

	// Cancel so the background sync goroutine does not outlive the test.
	if got.syncCancel != nil {
		got.syncCancel()
	}
}

// TestSeededBootSkipsSetup: a populated database must go straight to the board.
func TestSeededBootSkipsSetup(t *testing.T) {
	if m := boot(t, true, 120, 34); m.mode != modeBoard {
		t.Errorf("mode = %v with problems already stored, want modeBoard", m.mode)
	}
}

// TestEmptySearchDoesNotReenterSetup: setup is a first-run state, not an "no rows" state.
// A filter that matches nothing must never relaunch it.
func TestEmptySearchDoesNotReenterSetup(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("/"), key("z"), key("z"), key("z"), key("q"))

	if len(m.rows) != 0 {
		t.Fatalf("expected no matches, got %d", len(m.rows))
	}
	if m.mode == modeSetup {
		t.Error("an empty search result re-entered the first-run setup screen")
	}
}
