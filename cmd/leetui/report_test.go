package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
)

func caseResult(in, want, got string, judged, passed bool) runner.CaseResult {
	return runner.CaseResult{
		Case:   runner.TestCase{Input: in, Expected: want},
		Actual: got, Judged: judged, Passed: passed,
	}
}

// The exit code is the contract an editor branches on, so each outcome is pinned.
func TestRunExitCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  runner.Result
		want int
	}{
		{"all passed", runner.Result{Cases: []runner.CaseResult{
			caseResult("[2,7]\n9", "[0,1]", "[0,1]", true, true),
		}}, exitOK},

		{"wrong answer", runner.Result{Cases: []runner.CaseResult{
			caseResult("[2,7]\n9", "[0,1]", "[9,9]", true, false),
		}}, exitFailed},

		{"crashed", runner.Result{Cases: []runner.CaseResult{
			{Case: runner.TestCase{Input: "[]"}, Err: errors.New("boom")},
		}}, exitFailed},

		// A compile error is not a wrong answer — nothing ran, so nothing was judged.
		{"did not compile", runner.Result{CompileErr: "expected ';'"}, exitProblem},

		{"nothing to judge", runner.Result{Cases: []runner.CaseResult{
			caseResult("[2,7]\n9", "", "[0,1]", false, false),
		}}, exitOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if got := reportRun(&buf, "two-sum", tc.res); got != tc.want {
				t.Errorf("exit code %d, want %d\noutput:\n%s", got, tc.want, buf.String())
			}
		})
	}
}

// TestRunReportsTheDiff: a bare FAIL sends the reader looking for the three values that
// are the whole diagnosis.
func TestRunReportsTheDiff(t *testing.T) {
	var buf bytes.Buffer
	reportRun(&buf, "two-sum", runner.Result{Cases: []runner.CaseResult{
		caseResult("[2,7,11,15]\n9", "[0,1]", "[9,9]", true, false),
	}})

	out := buf.String()
	for _, want := range []string{"FAIL", "[2,7,11,15] · 9", "want  [0,1]", "got   [9,9]"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Multi-parameter input must stay on the line that labels it.
	if strings.Contains(out, "in    [2,7,11,15]\n9") {
		t.Errorf("input broke across lines:\n%s", out)
	}
}

// TestRunOutputHasNoEscapeCodes: this lands in an editor's message area as often as in a
// terminal, and neither renders a Departure Board.
func TestRunOutputHasNoEscapeCodes(t *testing.T) {
	var buf bytes.Buffer
	reportRun(&buf, "two-sum", runner.Result{Cases: []runner.CaseResult{
		caseResult("[2,7]\n9", "[0,1]", "[0,1]", true, true),
	}})
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("CLI output contains ANSI escape codes:\n%q", buf.String())
	}
}

// TestUncuratedMismatchDoesNotClaimAuthority: metaData cannot express in-place, unordered,
// or float-tolerant answers, so a mismatch on an uncurated problem might be the
// comparator's fault rather than the solution's (D-003).
func TestUncuratedMismatchDoesNotClaimAuthority(t *testing.T) {
	var buf bytes.Buffer
	slug := "a-problem-with-no-override"
	if runner.HasOverride(slug) {
		t.Fatalf("%s unexpectedly has a curated comparator", slug)
	}

	reportRun(&buf, slug, runner.Result{Cases: []runner.CaseResult{
		caseResult("[1]", "[1]", "[2]", true, false),
	}})

	if !strings.Contains(buf.String(), "no curated comparator") {
		t.Errorf("an uncurated mismatch was reported as a plain failure:\n%s", buf.String())
	}
}

func TestJudgementExitCodes(t *testing.T) {
	var buf bytes.Buffer
	accepted := leetcode.Judgement{
		StatusCode: leetcode.VerdictAccepted, StatusMsg: "Accepted",
		Runtime: "3 ms", RuntimePercentile: 91.2,
	}
	if got := reportJudgement(&buf, accepted); got != exitOK {
		t.Errorf("accepted exited %d, want %d", got, exitOK)
	}
	if !strings.Contains(buf.String(), "beats 91.2%") {
		t.Errorf("percentile missing from:\n%s", buf.String())
	}

	buf.Reset()
	wrong := leetcode.Judgement{
		StatusCode: leetcode.VerdictWrongAnswer, StatusMsg: "Wrong Answer",
		TotalCorrect: 3, TotalTestcases: 5,
		LastTestcase: "[3,3]\n6", ExpectedOutput: "[0,1]", CodeOutput: []string{"[1,0]"},
	}
	if got := reportJudgement(&buf, wrong); got != exitFailed {
		t.Errorf("wrong answer exited %d, want %d", got, exitFailed)
	}
	for _, want := range []string{"passed 3 of 5", "[3,3] · 6", "want  [0,1]", "got   [1,0]"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("missing %q in:\n%s", want, buf.String())
		}
	}
}
