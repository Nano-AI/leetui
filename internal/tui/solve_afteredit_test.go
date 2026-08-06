package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// loadDetail settles the selected problem's detail.
//
// boot runs before the statement is in the store, so the model holds no detail and every
// solve action would decline. This is what a real session gets for free by the time the
// user presses a key.
func loadDetail(t *testing.T, m Model) Model {
	t.Helper()
	var msgs []tea.Msg
	collect(m.loadDetailForCursor(), &msgs)
	return drive(t, m, msgs...)
}

// touchSolution rewrites the solution file with a later timestamp, standing in for a save
// made while leetui was suspended behind the editor.
func touchSolution(t *testing.T, ws workspace.Workspace, body string) {
	t.Helper()
	path := ws.Path(1, "two-sum", "solution.py")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}

// TestEditExitRunsTheTests is the fix for "running test cases after exiting a problem is
// annoying". The editor took over the terminal, so the watcher saw nothing while the
// editing happened; on the way back, the run fires without a keypress.
func TestEditExitRunsTheTests(t *testing.T) {
	m := boot(t, true, 120, 32)
	dir := t.TempDir()
	ws := prepared(t, m, dir)
	m.cfg.Workspace = dir

	// Select two-sum and let the detail land, so startRun has something to run.
	m = loadDetail(t, m)
	if m.detail == nil || m.detail.Slug != "two-sum" {
		t.Fatalf("expected two-sum selected, got %v", m.detail)
	}

	// First look records the timestamp; nothing has changed yet.
	if m.noteSolutionChanged() {
		t.Fatal("a first sighting reported a change")
	}
	touchSolution(t, ws, "class Solution:\n    def twoSum(self, nums, target):\n        return []\n")

	// Assert on the RESULT, not on m.running: the run completes inside drive, so the
	// flag is back to false by the time we look and would pass either way.
	m = drive(t, m, editDoneMsg{})
	if m.runResult == nil || m.runSlug != "two-sum" {
		t.Errorf("exiting the editor did not run the tests (result=%v slug=%q)",
			m.runResult, m.runSlug)
	}
}

// TestEditExitWithNoChangeRunsNothing: reopening a file and quitting without typing
// should not spend a compile.
func TestEditExitWithNoChangeRunsNothing(t *testing.T) {
	m := boot(t, true, 120, 32)
	dir := t.TempDir()
	prepared(t, m, dir)
	m.cfg.Workspace = dir

	m = loadDetail(t, m)
	if m.detail == nil {
		t.Fatal("no detail loaded; the test would pass for the wrong reason")
	}
	m.noteSolutionChanged() // record the current timestamp

	m = drive(t, m, editDoneMsg{})
	if m.runResult != nil {
		t.Error("an unchanged file still triggered a run")
	}
}

func TestRunAfterEditCanBeTurnedOff(t *testing.T) {
	m := boot(t, true, 120, 32)
	dir := t.TempDir()
	ws := prepared(t, m, dir)
	m.cfg.Workspace = dir
	m.cfg.RunAfterEdit = false

	m = loadDetail(t, m)
	if m.detail == nil {
		t.Fatal("no detail loaded; the test would pass for the wrong reason")
	}
	m.noteSolutionChanged()
	touchSolution(t, ws, "class Solution:\n    def twoSum(self, nums, target):\n        return []\n")

	m = drive(t, m, editDoneMsg{})
	if m.runResult != nil {
		t.Error("run_after_edit = false still ran the tests")
	}
}

// TestOneSaveRunsOnce guards the seam between the watcher and the after-edit run: both
// read the same timestamp, so whichever notices a save first claims it.
func TestOneSaveRunsOnce(t *testing.T) {
	m := boot(t, true, 120, 32)
	dir := t.TempDir()
	ws := prepared(t, m, dir)
	m.cfg.Workspace = dir

	m = loadDetail(t, m)
	if m.detail == nil {
		t.Fatal("no detail loaded; the test would pass for the wrong reason")
	}
	m.noteSolutionChanged()
	touchSolution(t, ws, "class Solution:\n    def twoSum(self, nums, target):\n        return []\n")

	// The exit claims the change.
	m = drive(t, m, editDoneMsg{})
	if m.runResult == nil {
		t.Fatal("the after-edit run did not happen")
	}

	// A watch tick arriving afterwards must find nothing new.
	if m.noteSolutionChanged() {
		t.Error("the same save was claimed twice — the tests would run again")
	}
}
