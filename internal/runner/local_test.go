package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const twoSumMeta = `{"name":"twoSum","params":[{"name":"nums","type":"integer[]"},
	{"name":"target","type":"integer"}],"return":{"type":"integer[]","size":2}}`

func python3(t *testing.T) *Local {
	t.Helper()
	l := NewLocal()
	lang, _ := Lookup("python3")
	if !l.Supports(lang) {
		t.Skip("python3 not on PATH")
	}
	return l
}

// generate lays out a problem folder with a solution and returns the directory.
func generate(t *testing.T, l *Local, slug, meta, solution string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "solution.py"), []byte(solution), 0o644); err != nil {
		t.Fatal(err)
	}
	lang, _ := Lookup("python3")
	p := Problem{Slug: slug, MetaData: meta}
	if err := l.Generate(context.Background(), p, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return dir
}

func run(t *testing.T, l *Local, dir, slug string, cases []TestCase) Result {
	t.Helper()
	lang, _ := Lookup("python3")
	res, err := l.Run(context.Background(), dir, lang, cases, RuleFor(slug))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

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

// TestGenerateRefusesDesignProblems: guessing a driver shape for a class-with-operations
// problem would produce confidently wrong answers.
func TestGenerateRefusesDesignProblems(t *testing.T) {
	l := python3(t)
	lang, _ := Lookup("python3")
	design := `{"classname":"LRUCache","methods":[{"name":"get"}]}`

	err := l.Generate(context.Background(), Problem{Slug: "lru-cache", MetaData: design}, lang, t.TempDir())
	if err == nil {
		t.Fatal("generated a driver for a design problem")
	}
}

func TestSupportsAndToolchain(t *testing.T) {
	l := NewLocal()

	java, _ := Lookup("java")
	if l.Supports(java) {
		t.Error("java has no vendored driver and must not report as locally runnable")
	}
	if l.MissingToolchain(java) != "" {
		t.Error("a language with no driver should not blame a missing toolchain")
	}

	// Rust has no driver yet, so it must report as judge-only rather than blaming a
	// missing toolchain — "install rustup" would be the wrong advice.
	rust, _ := Lookup("rust")
	if rust.Local {
		t.Error("rust is marked local but has no driver in drivers/")
	}
	if l.MissingToolchain(rust) != "" {
		t.Errorf("MissingToolchain(rust) = %q; a driverless language must not blame a toolchain",
			l.MissingToolchain(rust))
	}

	// Python has a driver, so on a machine with python3 it must actually run.
	py, _ := Lookup("python3")
	if !py.Local {
		t.Error("python3 has a vendored driver and should be marked local")
	}
}
