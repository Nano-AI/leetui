package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunPassingSolution(t *testing.T) {
	l := python3(t)
	dir := generate(t, l, "two-sum", twoSumMeta, `
class Solution:
    def twoSum(self, nums, target):
        seen = {}
        for i, n in enumerate(nums):
            if target - n in seen:
                return [seen[target - n], i]
            seen[n] = i
`)

	res := run(t, l, dir, "two-sum", []TestCase{
		{Input: "[2,7,11,15]\n9", Expected: "[0,1]"},
		{Input: "[3,2,4]\n6", Expected: "[1,2]"},
	})

	if !res.Passed() {
		t.Errorf("expected a pass, got %+v", res.Cases)
	}
	passed, failed, errored := res.Summary()
	if passed != 2 || failed != 0 || errored != 0 {
		t.Errorf("summary = %d/%d/%d, want 2/0/0", passed, failed, errored)
	}
}

func TestRunWrongAnswerIsDataNotError(t *testing.T) {
	l := python3(t)
	dir := generate(t, l, "two-sum", twoSumMeta, `
class Solution:
    def twoSum(self, nums, target):
        return [9, 9]
`)

	res := run(t, l, dir, "two-sum", []TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}})
	if res.Passed() {
		t.Fatal("a wrong answer reported as passing")
	}
	if res.Cases[0].Err != nil {
		t.Errorf("a wrong answer must not be an error: %v", res.Cases[0].Err)
	}
	if res.Cases[0].Actual != "[9,9]" {
		t.Errorf("actual = %q, want [9,9]", res.Cases[0].Actual)
	}
}

// TestRunCrashIsReportedPerCase: a solution that throws must fail one case with a
// readable traceback, not abort the run.
func TestRunCrashIsReportedPerCase(t *testing.T) {
	l := python3(t)
	dir := generate(t, l, "two-sum", twoSumMeta, `
class Solution:
    def twoSum(self, nums, target):
        raise ValueError("boom")
`)

	res := run(t, l, dir, "two-sum", []TestCase{
		{Input: "[2,7,11,15]\n9", Expected: "[0,1]"},
		{Input: "[3,2,4]\n6", Expected: "[1,2]"},
	})
	if len(res.Cases) != 2 {
		t.Fatalf("ran %d cases, want both", len(res.Cases))
	}
	for i, c := range res.Cases {
		if c.Err == nil {
			t.Errorf("case %d did not report the crash", i)
			continue
		}
		if !strings.Contains(c.Err.Error(), "boom") {
			t.Errorf("case %d error lost the exception: %v", i, c.Err)
		}
	}
}

// TestRunTimeout: an infinite loop must fail its case rather than hang the app.
func TestRunTimeout(t *testing.T) {
	l := python3(t)
	l.Timeout = 900 * time.Millisecond

	dir := generate(t, l, "two-sum", twoSumMeta, `
class Solution:
    def twoSum(self, nums, target):
        while True:
            pass
`)

	started := time.Now()
	res := run(t, l, dir, "two-sum", []TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}})
	elapsed := time.Since(started)

	if res.Cases[0].Err == nil {
		t.Fatal("an infinite loop did not fail its case")
	}
	if !strings.Contains(res.Cases[0].Err.Error(), "timed out") {
		t.Errorf("error = %v, want a timeout", res.Cases[0].Err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("timeout took %s to fire", elapsed)
	}
}

// TestInPlaceProblem is the metaData blind spot: the answer is the mutated argument,
// and the return value is only a length.
func TestInPlaceProblem(t *testing.T) {
	l := python3(t)
	const meta = `{"name":"removeDuplicates","params":[{"name":"nums","type":"integer[]"}],
		"return":{"type":"integer"}}`

	dir := generate(t, l, "remove-duplicates-from-sorted-array", meta, `
class Solution:
    def removeDuplicates(self, nums):
        k = 0
        for n in nums:
            if k == 0 or nums[k-1] != n:
                nums[k] = n
                k += 1
        return k
`)

	res := run(t, l, dir, "remove-duplicates-from-sorted-array",
		[]TestCase{{Input: "[1,1,2]", Expected: "[1,2]"}})

	if !res.Passed() {
		t.Errorf("in-place answer not judged correctly: actual=%q err=%v",
			res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestUnjudgedCaseReportsOutput(t *testing.T) {
	l := python3(t)
	dir := generate(t, l, "two-sum", twoSumMeta, `
class Solution:
    def twoSum(self, nums, target):
        return [0, 1]
`)

	// No Expected: the user added this case by hand and wants to see the output.
	res := run(t, l, dir, "two-sum", []TestCase{{Input: "[2,7,11,15]\n9"}})
	if res.Cases[0].Judged {
		t.Error("a case with no expectation was judged")
	}
	if res.Cases[0].Actual != "[0,1]" {
		t.Errorf("actual = %q", res.Cases[0].Actual)
	}
}

// TestGenerateHandlesDesignProblems: Python's driver implements the
// class-with-operations shape, so it generates rather than declines. Languages without
// that shape are covered by TestDesignDeclinedForLanguagesWithoutIt.
func TestGenerateHandlesDesignProblems(t *testing.T) {
	l := python3(t)
	lang, _ := Lookup("python3")
	design := `{"classname":"LRUCache",
		"constructor":{"params":[{"name":"capacity","type":"integer"}]},
		"methods":[{"name":"get","params":[{"name":"key","type":"integer"}],
		            "return":{"type":"integer"}}]}`

	if err := l.Generate(context.Background(),
		Problem{Slug: "lru-cache", MetaData: design}, lang, t.TempDir()); err != nil {
		t.Fatalf("declined a design problem Python can run: %v", err)
	}
}
