package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
)

func TestEditorResolution(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	// Config wins, so leetui can be pointed somewhere without touching the shell.
	if got := resolveEditor("nvim"); !strings.HasSuffix(got.Command, "nvim") {
		t.Errorf("configured editor = %q, want nvim", got.Command)
	}

	// An unknown command is honoured verbatim rather than overridden.
	if got := resolveEditor("/opt/weird/ed"); got.Command != "/opt/weird/ed" {
		t.Errorf("unknown editor = %q, want it used as written", got.Command)
	}

	// Then the environment, with VISUAL outranking EDITOR.
	t.Setenv("EDITOR", "some-editor")
	if got := resolveEditor(""); got.Command != "some-editor" {
		t.Errorf("$EDITOR = %q", got.Command)
	}
	t.Setenv("VISUAL", "other-editor")
	if got := resolveEditor(""); got.Command != "other-editor" {
		t.Errorf("$VISUAL should outrank $EDITOR, got %q", got.Command)
	}

	// With nothing set, the fallback must still name something runnable — both vars are
	// commonly unset (D-012).
	os.Unsetenv("VISUAL")
	os.Unsetenv("EDITOR")
	if got := resolveEditor(""); got.Command == "" {
		t.Error("fallback produced no editor")
	}

	// A GUI editor must never lose its blocking flag on the way through.
	if e, ok := editor.Lookup("code"); ok {
		if _, args := e.Launch("/tmp/x.py"); len(args) < 2 || args[0] != "--wait" {
			t.Errorf("VS Code launched without --wait: %v", args)
		}
	}
}

// TestPickerListsOnlyOfferedLanguages: LeetCode does not offer every language on every
// problem, so the picker is built from the problem's own snippet table.
func TestPickerListsOnlyOfferedLanguages(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.detail = nil
	if len(m.pickerLangs()) < 10 {
		t.Error("with no problem selected the picker should offer everything")
	}

	m = drive(t, m, key("j")) // select something
	m.detail = detailWithSnippets("python3", "golang", "java")
	got := m.pickerLangs()
	if len(got) != 3 {
		t.Fatalf("picker offered %d languages, want 3", len(got))
	}
	// Locally-runnable languages sort first, so the cursor lands near the fast loop.
	// Asserted as an invariant rather than a fixed count: which languages have drivers
	// changes as they are written.
	seenRemote := false
	for _, l := range got {
		if !l.Local {
			seenRemote = true
			continue
		}
		if seenRemote {
			t.Errorf("%s runs locally but sorted after a judge-only language", l.Slug)
		}
	}
}

func TestPickerSelectsLanguage(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.picking = pickLang
	m.pickIdx = 0

	langs := m.pickerLangs()
	if len(langs) == 0 {
		t.Skip("no languages")
	}

	m = drive(t, m, key("enter"))
	if m.picking != pickNone {
		t.Error("picker stayed open after choosing")
	}
	if m.lang.Slug != langs[0].Slug {
		t.Errorf("chose %q, want %q", m.lang.Slug, langs[0].Slug)
	}
}

// TestRunSummaryHedgesWhenUncurated is the D-003 promise: a local mismatch on a problem
// with no curated comparator must not be stated as a wrong answer.
func TestRunSummaryHedgesWhenUncurated(t *testing.T) {
	failing := runner.Result{Cases: []runner.CaseResult{
		{Judged: true, Passed: false},
		{Judged: true, Passed: true},
	}}

	uncurated := runSummary("a-problem-nobody-curated", failing)
	if !strings.Contains(uncurated, "judge") {
		t.Errorf("uncurated mismatch should point at the judge, got %q", uncurated)
	}

	// "subsets" IS curated (unordered), so a failure there can be stated plainly.
	curated := runSummary("subsets", failing)
	if strings.Contains(curated, "no curated") {
		t.Errorf("curated problem should not hedge, got %q", curated)
	}

	passing := runner.Result{Cases: []runner.CaseResult{{Judged: true, Passed: true}}}
	if got := runSummary("two-sum", passing); !strings.Contains(got, "passed") {
		t.Errorf("passing summary = %q", got)
	}
}

func TestJudgeSummary(t *testing.T) {
	accepted := leetcode.Judgement{
		State: "SUCCESS", StatusCode: leetcode.VerdictAccepted,
		Runtime: "58 ms", RuntimePercentile: 91.2,
	}
	got := judgeSummary(accepted)
	for _, want := range []string{"Accepted", "58 ms", "91.2"} {
		if !strings.Contains(got, want) {
			t.Errorf("accepted summary %q missing %q", got, want)
		}
	}

	wrong := leetcode.Judgement{
		State: "SUCCESS", StatusCode: leetcode.VerdictWrongAnswer,
		TotalCorrect: 34, TotalTestcases: 60, LastTestcase: "[1,2]\n3",
	}
	got = judgeSummary(wrong)
	for _, want := range []string{"Wrong answer", "34 of 60"} {
		if !strings.Contains(got, want) {
			t.Errorf("wrong-answer summary %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "\n") {
		t.Error("the failing case must be flattened onto one status line")
	}

	compile := leetcode.Judgement{
		State: "SUCCESS", StatusCode: leetcode.VerdictCompileError,
		CompileError: "line 4: expected ':'\n  more context",
	}
	if got := judgeSummary(compile); !strings.Contains(got, "expected ':'") {
		t.Errorf("compile summary = %q", got)
	}
}

func TestVerdictMapping(t *testing.T) {
	cases := map[leetcode.VerdictCode]string{
		leetcode.VerdictAccepted:            "accepted",
		leetcode.VerdictWrongAnswer:         "wrong answer",
		leetcode.VerdictTimeLimitExceeded:   "time limit exceeded",
		leetcode.VerdictMemoryLimitExceeded: "memory limit exceeded",
		leetcode.VerdictCompileError:        "compile error",
		leetcode.VerdictRuntimeError:        "runtime error",
	}
	for code, want := range cases {
		got := verdictOf(leetcode.Judgement{StatusCode: code})
		if got.Text() != want {
			t.Errorf("verdict %d = %q, want %q", code, got.Text(), want)
		}
		if !got.Resolved() {
			t.Errorf("verdict %d should be resolved", code)
		}
	}
}

// detailWithSnippets builds a Detail offering exactly the given languages.
func detailWithSnippets(slugs ...string) *store.Detail {
	d := &store.Detail{Snippets: map[string]string{}}
	d.Slug = "test-problem"
	for _, s := range slugs {
		d.Snippets[s] = "stub"
	}
	return d
}
