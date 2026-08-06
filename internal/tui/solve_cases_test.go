package tui

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/workspace"
)

const twoSumMeta = `{"name":"twoSum","params":[{"name":"nums","type":"integer[]"},
	{"name":"target","type":"integer"}],"return":{"type":"integer[]","size":2}}`

// twoSumHTML is what LeetCode actually serves for question.content: the expected answers
// live in the statement prose, inside <pre> blocks, and nowhere else.
const twoSumHTML = `<p>Given an array of integers <code>nums</code> and an integer <code>target</code>, return indices of the two numbers such that they add up to <code>target</code>.</p>

<p><strong class="example">Example 1:</strong></p>

<pre>
<strong>Input:</strong> nums = [2,7,11,15], target = 9
<strong>Output:</strong> [0,1]
<strong>Explanation:</strong> Because nums[0] + nums[1] == 9, we return [0, 1].
</pre>

<p><strong class="example">Example 2:</strong></p>

<pre>
<strong>Input:</strong> nums = [3,2,4], target = 6
<strong>Output:</strong> [1,2]
</pre>`

// prepared runs the solve loop's file layout for two-sum in dir and returns the
// workspace. The caller owns dir so a test can seed a file into it first.
func prepared(t *testing.T, m Model, dir string) workspace.Workspace {
	t.Helper()
	ctx := context.Background()

	err := m.store.SetDetail(ctx, &leetcode.Problem{
		QuestionID: "1", FrontendID: "1", Title: "Two Sum", Slug: "two-sum",
		Content:          twoSumHTML,
		MetaData:         twoSumMeta,
		ExampleTestcases: "[2,7,11,15]\n9\n[3,2,4]\n6",
		Snippets: []leetcode.CodeSnippet{{
			Lang: "Python3", LangSlug: "python3",
			Code: "class Solution:\n    def twoSum(self, nums, target):\n        pass\n",
		}},
	})
	if err != nil {
		t.Fatalf("seed detail: %v", err)
	}

	d, err := m.store.Get(ctx, "two-sum")
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}

	m.cfg.Workspace = dir
	lang, _ := runner.Lookup("python3")
	if _, _, err := m.prepare(ctx, d, lang); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	return ws
}

// TestPreparedCasesCarryExpectedAnswers is the regression guard for "none had an expected
// answer to check against". Expected values are scraped from the statement, and feeding
// the scraper the terminal-rendered statement instead of markdown silently produced test
// cases with no answers — a local run that could only ever print output, never judge it.
func TestPreparedCasesCarryExpectedAnswers(t *testing.T) {
	m := boot(t, true, 120, 32)
	ws := prepared(t, m, t.TempDir())

	raw, err := ws.ReadFile(1, "two-sum", workspace.TestcasesFile)
	if err != nil {
		t.Fatalf("read testcases: %v", err)
	}

	cases := runner.LoadCases(raw)
	if len(cases) != 2 {
		t.Fatalf("wrote %d cases, want 2:\n%s", len(cases), raw)
	}
	want := []string{"[0,1]", "[1,2]"}
	for i, c := range cases {
		if c.Expected != want[i] {
			t.Errorf("case %d expected %q, want %q\nfile:\n%s", i+1, c.Expected, want[i], raw)
		}
	}
	if cases[0].Input != "[2,7,11,15]\n9" {
		t.Errorf("case 1 input is %q", cases[0].Input)
	}
}

// TestReadmeIsMarkdownNotTerminalOutput guards the other half of the same mistake: the
// README is a file people read in an editor and on GitHub, so escape codes in it are
// garbage rather than colour.
func TestReadmeIsMarkdownNotTerminalOutput(t *testing.T) {
	m := boot(t, true, 120, 32)
	ws := prepared(t, m, t.TempDir())

	readme, err := ws.ReadFile(1, "two-sum", workspace.ReadmeFile)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if strings.Contains(readme, "\x1b[") {
		t.Errorf("README contains ANSI escape codes:\n%q", readme[:minInt(len(readme), 400)])
	}
	if !strings.Contains(readme, "Output:") {
		t.Errorf("README lost the example blocks:\n%s", readme)
	}
}

// TestStaleTestcasesAreRepaired covers workspaces written before the fix: the file is
// there, so createIfMissing leaves it, and every run would keep reporting no expected
// answers forever.
func TestStaleTestcasesAreRepaired(t *testing.T) {
	m := boot(t, true, 120, 32)

	dir := t.TempDir()
	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	// What the broken version wrote: inputs, and an empty output for every case.
	if _, err := ws.WriteTestcases(1, "two-sum", "[2,7,11,15]\n9\noutput:\n\n\n[3,2,4]\n6\noutput:\n\n"); err != nil {
		t.Fatalf("seed stale file: %v", err)
	}

	ws = prepared(t, m, dir)

	cases := runner.LoadCases(mustRead(t, ws, workspace.TestcasesFile))
	if len(cases) == 0 || cases[0].Expected == "" {
		t.Fatalf("stale test cases were not repaired: %+v", cases)
	}
}

// TestHandWrittenTestcasesSurvive is the other side of that repair: a file the user
// edited must never be rewritten, even when leetui thinks it knows better.
func TestHandWrittenTestcasesSurvive(t *testing.T) {
	m := boot(t, true, 120, 32)

	dir := t.TempDir()
	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	mine := "[1,2,3]\n5\noutput:\n[0,2]\n"
	if _, err := ws.WriteTestcases(1, "two-sum", mine); err != nil {
		t.Fatalf("seed hand-written file: %v", err)
	}

	ws = prepared(t, m, dir)

	if got := mustRead(t, ws, workspace.TestcasesFile); got != mine {
		t.Errorf("hand-written cases were overwritten:\ngot:\n%s\nwant:\n%s", got, mine)
	}
}

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

func mustRead(t *testing.T, ws workspace.Workspace, name string) string {
	t.Helper()
	s, err := ws.ReadFile(1, "two-sum", name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return s
}
