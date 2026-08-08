package tui

import (
	"context"
	"os"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// TestLocalRunJudgesTheCases is the end-to-end shape of the reported bug: cases with no
// expected answers still RUN, so nothing errors — the run just comes back saying there
// was nothing to check. This asserts a real pass and a real fail instead.
func TestLocalRunJudgesTheCases(t *testing.T) {
	engine := runner.NewLocal()
	lang, _ := runner.Lookup("python3")
	if !engine.Supports(lang) {
		t.Skip("python3 not on PATH")
	}

	for _, tc := range []struct {
		name, body string
		wantPass   bool
	}{
		{"correct", `        seen = {}
        for i, n in enumerate(nums):
            if target - n in seen:
                return [seen[target - n], i]
            seen[n] = i
`, true},
		{"wrong", "        return [9, 9]\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := boot(t, true, 120, 32)
			dir := t.TempDir()
			ws := prepared(t, m, dir)

			solution := "class Solution:\n    def twoSum(self, nums, target):\n" + tc.body
			path := ws.Path(1, "two-sum", lang.Filename())
			if err := os.WriteFile(path, []byte(solution), 0o644); err != nil {
				t.Fatalf("write solution: %v", err)
			}

			ctx := context.Background()
			problemDir := ws.Dir(1, "two-sum")
			p := runner.Problem{Slug: "two-sum", Title: "Two Sum", MetaData: twoSumMeta}
			if err := engine.Generate(ctx, p, lang, problemDir); err != nil {
				t.Fatalf("generate: %v", err)
			}

			cases := runner.LoadCases(mustRead(t, ws, workspace.TestcasesFile))
			if !runner.HasExpected(cases) {
				t.Fatal("cases have no expected answers — this is the bug")
			}

			res, err := engine.Run(ctx, problemDir, lang, cases, runner.RuleFor("two-sum"))
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if len(res.Cases) != 2 {
				t.Fatalf("ran %d cases, want 2", len(res.Cases))
			}
			for i, c := range res.Cases {
				if !c.Judged {
					t.Errorf("case %d ran but was not judged — no expected answer", i+1)
					continue
				}
				if c.Passed != tc.wantPass {
					t.Errorf("case %d passed=%v, want %v (got %q, expected %q)",
						i+1, c.Passed, tc.wantPass, c.Actual, c.Case.Expected)
				}
			}
		})
	}
}

// TestSubmitShowsTheQueueOverAStaleRun is the reported bug: after running locally,
// pressing submit appeared to do nothing. The side pane preferred any existing run
// result unconditionally, so the verdict landed in a queue that was never drawn.
func TestSubmitShowsTheQueueOverAStaleRun(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	// A local run has happened and its result is on screen.
	m.runSlug = m.rows[m.cursor].Slug
	m.runResult = &runner.Result{Cases: []runner.CaseResult{
		{Judged: true, Passed: true, Actual: "3"},
	}}
	if !m.showRun() {
		t.Fatal("a fresh run result should own the pane")
	}

	// Submitting claims the pane, even though the run result is still there.
	m.queueOnTop = true
	if m.showRun() {
		t.Error("the run result still covers the pane after submitting")
	}

	// And running claims it back.
	m.queueOnTop = false
	if !m.showRun() {
		t.Error("running did not take the pane back")
	}
}
