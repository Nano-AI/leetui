package main

import (
	"flag"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
)

func TestFlagsTerminatorPreservesRemainingArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--", "two-sum", "--note", "reason"},
		{"two-sum", "--", "3sum", "--note", "reason"},
	} {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		note := fs.String("note", "", "")
		got, err := parseFlags(fs, args)
		var want []string
		for _, arg := range args {
			if arg != "--" {
				want = append(want, arg)
			}
		}
		if err != nil || !reflect.DeepEqual(got, want) || *note != "" {
			t.Fatalf("parseFlags(%q) = %q, note=%q, err=%v", args, got, *note, err)
		}
	}
	// A terminator-shaped flag value must still be consumed as a value.
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	note := fs.String("note", "", "")
	got, err := parseFlags(fs, []string{"--note", "--", "two-sum"})
	if err != nil || *note != "--" || !reflect.DeepEqual(got, []string{"two-sum"}) {
		t.Fatalf("flag value: %v %q %v", got, *note, err)
	}
}

func TestReportRunDoesNotClaimUncheckedCasesPassed(t *testing.T) {
	for _, cases := range [][]runner.CaseResult{
		nil,
		{{Actual: "42"}},
		{{Judged: true, Passed: true}, {Actual: "42"}},
	} {
		var out strings.Builder
		if code := reportRun(&out, "two-sum", runner.Result{Cases: cases}); code != exitProblem {
			t.Errorf("unchecked run returned %d, want 2: %s", code, out.String())
		}
		if strings.Contains(out.String(), "All ") {
			t.Errorf("false success: %s", out.String())
		}
	}
}

func TestCommandsRejectExtraArgumentsBeforeUsingApp(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*app, []string) (int, error)
		args []string
	}{
		{"pull", runPull, []string{"two-sum", "typo"}},
		{"run", runRun, []string{"two-sum", "typo"}},
		{"path", runPath, []string{"two-sum", "typo"}},
		{"submit", runSubmit, []string{"two-sum", "typo"}},
		{"contest show", runContest, []string{"weekly-test", "typo"}},
		{"contest pull", runContest, []string{"pull", "weekly-test", "typo"}},
		{"contest submit", runContest, []string{"submit", "weekly-test", "two-sum", "typo"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalid arguments reached app dependencies: %v", r)
				}
			}()
			if code, err := tc.run(nil, tc.args); code != exitProblem || err == nil {
				t.Errorf("invalid arguments returned %d, %v", code, err)
			}
		})
	}
}
