package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
)

// benchLines returns the workbench strip's rendered lines, stripped of styling.
func benchLines(t *testing.T, m Model, w int) []string {
	t.Helper()
	return strings.Split(stripANSI(m.viewWorkbench(w)), "\n")
}

// TestWorkbenchNamesTheFile is the requirement the whole strip exists for: when the editing
// happens in another pane, leetui must say WHERE. Without a path there is nothing to open.
func TestWorkbenchNamesTheFile(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.Workspace = t.TempDir()

	out := stripANSI(m.viewWorkbench(50))
	if !strings.Contains(out, "0001-two-sum") {
		t.Errorf("the strip does not name the problem folder:\n%s", out)
	}
	if !strings.Contains(out, "solution.py") {
		t.Errorf("the strip does not name the solution file:\n%s", out)
	}
}

// TestWorkbenchPathMatchesReality guards the one thing this strip must never do: show a
// path that is not where the file actually goes.
func TestWorkbenchPathMatchesReality(t *testing.T) {
	m := boot(t, true, 120, 34)
	dir := t.TempDir()
	m.cfg.Workspace = dir

	shown := m.solutionPath()
	// Prepare is what actually creates it, so the two must agree.
	ws := prepared(t, m, dir)
	real := ws.Path(1, "two-sum", "solution.py")

	if shown != real {
		t.Errorf("the strip shows %q but the file is at %q", shown, real)
	}
	if _, err := os.Stat(real); err != nil {
		t.Errorf("prepare did not create the file the strip pointed at: %v", err)
	}
}

func TestWorkbenchStates(t *testing.T) {
	m := boot(t, true, 120, 34)
	dir := t.TempDir()
	m.cfg.Workspace = dir

	// Nothing on disk yet: say so rather than showing a path with no explanation.
	if got := stripANSI(m.workbenchState(m.solutionPath())); !strings.Contains(got, "not created") {
		t.Errorf("state before creation = %q, want it to say the file is not there", got)
	}

	prepared(t, m, dir)
	if got := stripANSI(m.workbenchState(m.solutionPath())); !strings.Contains(got, "watching") {
		t.Errorf("state after creation = %q, want watching", got)
	}

	// Watching off must not claim otherwise.
	off := m
	off.cfg.WatchSolution = false
	if got := stripANSI(off.workbenchState(off.solutionPath())); strings.Contains(got, "watching") {
		t.Errorf("watch_solution = false still reported %q", got)
	}
}

// TestWorkbenchDoesNotPromiseARunItCannotDo: a language with no local driver has nothing
// to re-run on save, so saying "watching" would promise something that never happens.
func TestWorkbenchDoesNotPromiseARunItCannotDo(t *testing.T) {
	m := boot(t, true, 120, 34)
	dir := t.TempDir()
	m.cfg.Workspace = dir
	prepared(t, m, dir)

	java, ok := runner.Lookup("java")
	if !ok {
		t.Fatal("java is not a known language")
	}
	m.lang = java

	// Put a file there so the state is not simply "not created".
	path := m.solutionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("class Solution {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := stripANSI(m.workbenchState(path))
	if strings.Contains(got, "watching") {
		t.Errorf("promised a re-run for a judge-only language: %q", got)
	}
	if !strings.Contains(got, "judge only") {
		t.Errorf("state = %q, want it to say judge only", got)
	}
}

// TestWorkbenchPathShortensFromTheLeft: the filename identifies the file; the middle of a
// long workspace root is what can be spared.
func TestWorkbenchPathShortensFromTheLeft(t *testing.T) {
	long := "/a/very/long/workspace/root/that/keeps/going/0001-two-sum/solution.py"

	got := fitPath(long, 30)
	if len([]rune(got)) > 30 {
		t.Errorf("fitPath returned %d cells for a 30 budget: %q", len([]rune(got)), got)
	}
	if !strings.HasSuffix(got, "solution.py") {
		t.Errorf("fitPath dropped the filename: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("fitPath truncated from the wrong end: %q", got)
	}
}

func TestWorkbenchFitsItsFrame(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.Workspace = t.TempDir()

	for _, w := range []int{40, 46, 60, 80} {
		for i, line := range benchLines(t, m, w) {
			if line == "" {
				continue
			}
			if got := len([]rune(line)); got > w {
				t.Errorf("width %d: line %d is %d cells: %q", w, i, got, line)
			}
		}
	}
}
