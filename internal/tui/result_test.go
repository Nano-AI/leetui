package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/runner"
)

func withResult(t *testing.T, r runner.Result) Model {
	t.Helper()
	m := boot(t, true, 120, 34)
	m.runResult = &r
	m.runSlug = m.rows[m.cursor].Slug
	return m
}

// TestResultShowsTheDiff is the whole point of the panel: a failure must show what
// differed, not just that something did.
func TestResultShowsTheDiff(t *testing.T) {
	m := withResult(t, runner.Result{Cases: []runner.CaseResult{
		{Case: runner.TestCase{Input: "[2,7,11,15]\n9", Expected: "[0,1]"},
			Actual: "[9,9]", Judged: true, Passed: false},
	}})

	out := stripANSI(m.viewResult(48, 16))
	t.Logf("\n%s", m.viewResult(48, 16))

	for _, want := range []string{"FAIL", "[2,7,11,15]", "[0,1]", "[9,9]"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff is missing %q:\n%s", want, out)
		}
	}
}

func TestResultCollapsesPasses(t *testing.T) {
	m := withResult(t, runner.Result{Cases: []runner.CaseResult{
		{Case: runner.TestCase{Input: "[1]", Expected: "[0]"}, Actual: "[0]", Judged: true, Passed: true},
		{Case: runner.TestCase{Input: "[2]", Expected: "[1]"}, Actual: "[1]", Judged: true, Passed: true},
	}})

	out := stripANSI(m.viewResult(48, 16))
	if strings.Contains(out, "want") {
		t.Errorf("a passing case was expanded:\n%s", out)
	}
	// Counted per line: the bezel tally ("2 PASSED") also contains "PASS".
	n := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "case ") && strings.Contains(line, "PASS") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("expected two collapsed PASS lines, got %d:\n%s", n, out)
	}
}

func TestResultShowsCrashAndCompileError(t *testing.T) {
	crash := withResult(t, runner.Result{Cases: []runner.CaseResult{
		{Case: runner.TestCase{Input: "[1]"}, Err: errors.New("ValueError: boom")},
	}})
	if out := stripANSI(crash.viewResult(48, 16)); !strings.Contains(out, "CRASHED") ||
		!strings.Contains(out, "boom") {
		t.Errorf("crash not surfaced:\n%s", out)
	}

	compile := withResult(t, runner.Result{CompileErr: "line 4: expected ':'"})
	if out := stripANSI(compile.viewResult(48, 16)); !strings.Contains(out, "compile") &&
		!strings.Contains(out, "expected ':'") {
		t.Errorf("compile error not surfaced:\n%s", out)
	}
}

// TestResultOnlyShowsForTheSelectedProblem: a result from elsewhere invites reading the
// wrong failure.
func TestResultOnlyShowsForTheSelectedProblem(t *testing.T) {
	m := withResult(t, runner.Result{Cases: []runner.CaseResult{{Judged: true, Passed: true}}})
	if !m.hasResult() {
		t.Fatal("result for the selected problem was hidden")
	}

	m.runSlug = "some-other-problem"
	if m.hasResult() {
		t.Error("a result from another problem was shown")
	}
	if strings.Contains(stripANSI(m.viewSidePane(48, 16)), "PASS") {
		t.Error("the side pane rendered a foreign result")
	}
}

// TestResultFitsItsPane guards the frame: long arrays must wrap, never overflow.
func TestResultFitsItsPane(t *testing.T) {
	long := "[" + strings.Repeat("123456,", 40) + "0]"
	m := withResult(t, runner.Result{Cases: []runner.CaseResult{
		{Case: runner.TestCase{Input: long, Expected: long}, Actual: "[]", Judged: true},
	}})

	for _, w := range []int{40, 48, 60} {
		for i, line := range strings.Split(m.viewResult(w, 18), "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Errorf("width %d: line %d is %d cols", w, i, got)
			}
		}
	}
}
