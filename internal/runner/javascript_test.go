package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// JavaScript and TypeScript share one driver and one binary: Node strips TypeScript's
// types itself, so solution.ts runs the way solution.js does. These tests run both
// through the same cases to prove that rather than assume it.

func nodeLang(t *testing.T, slug string) (*Local, Lang) {
	t.Helper()
	l := NewLocal()
	lang, _ := Lookup(slug)
	if !l.Supports(lang) {
		t.Skip("node not on PATH")
	}
	return l, lang
}

// generateJSDir lays out a folder with a raw solution file and generates the entry.
//
// The solution is written unscaffolded: what matters here is that the driver finds a
// bare `var twoSum = function(...)` with no exports, which is exactly what LeetCode's
// starter is and the reason the driver concatenates rather than imports.
func generateJSDir(t *testing.T, l *Local, lang Lang, slug, meta, solution string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, lang.Filename()), []byte(solution), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Problem{Slug: slug, MetaData: meta}
	if err := l.Generate(context.Background(), p, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return dir
}

func runJSCases(t *testing.T, l *Local, lang Lang, dir, slug string, cases []TestCase) Result {
	t.Helper()
	res, err := l.Run(context.Background(), dir, lang, cases, RuleFor(slug))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

func TestJavaScriptRunsAPlainFunction(t *testing.T) {
	l, lang := nodeLang(t, "javascript")
	dir := generateJSDir(t, l, lang, "two-sum", twoSumMeta, `
var twoSum = function(nums, target) {
    const seen = new Map();
    for (let i = 0; i < nums.length; i++) {
        if (seen.has(target - nums[i])) return [seen.get(target - nums[i]), i];
        seen.set(nums[i], i);
    }
    return [];
};
`)

	res := runJSCases(t, l, lang, dir, "two-sum", []TestCase{
		{Input: "[2,7,11,15]\n9", Expected: "[0,1]"},
		{Input: "[3,2,4]\n6", Expected: "[1,2]"},
	})
	if !res.Passed() {
		t.Errorf("expected a pass, got %+v", res.Cases)
	}
}

// TestTypeScriptNeedsNoTranspiler is the reason JS and TS are one language here. If Node
// ever stops stripping types this fails, which is the signal to add a transpile step
// rather than discover it from a user.
func TestTypeScriptNeedsNoTranspiler(t *testing.T) {
	l, lang := nodeLang(t, "typescript")
	dir := generateJSDir(t, l, lang, "two-sum", twoSumMeta, `
function twoSum(nums: number[], target: number): number[] {
    const seen = new Map<number, number>();
    for (let i = 0; i < nums.length; i++) {
        const want: number = target - nums[i];
        if (seen.has(want)) return [seen.get(want)!, i];
        seen.set(nums[i], i);
    }
    return [];
}
`)

	res := runJSCases(t, l, lang, dir, "two-sum", []TestCase{
		{Input: "[2,7,11,15]\n9", Expected: "[0,1]"},
	})
	if !res.Passed() {
		t.Errorf("typescript did not run: %+v", res.Cases)
	}
}

// TestJavaScriptEmptyListPrintsBrackets is the case that found a real bug. An empty list
// and a null integer are both `null` in JavaScript, but the judge prints `[]` for one and
// `null` for the other — so the driver has to be told the declared return type.
func TestJavaScriptEmptyListPrintsBrackets(t *testing.T) {
	l, lang := nodeLang(t, "javascript")
	const meta = `{"name":"reverseList","params":[{"name":"head","type":"ListNode"}],"return":{"type":"ListNode"}}`

	dir := generateJSDir(t, l, lang, "reverse-linked-list", meta, `
var reverseList = function(head) {
    let prev = null;
    while (head) { const nx = head.next; head.next = prev; prev = head; head = nx; }
    return prev;
};
`)

	res := runJSCases(t, l, lang, dir, "reverse-linked-list", []TestCase{
		{Input: "[1,2,3]", Expected: "[3,2,1]"},
		{Input: "[]", Expected: "[]"},
	})
	for i, c := range res.Cases {
		if !c.Passed {
			t.Errorf("case %d: got %q, want %q", i+1, c.Actual, c.Case.Expected)
		}
	}
}

// TestJavaScriptRunsADesignProblem — Go and C++ decline these because they would have to
// reconstruct type declarations. JavaScript does not: a design problem is a constructor
// function and its prototype, which is what the starter already gives you.
func TestJavaScriptRunsADesignProblem(t *testing.T) {
	l, lang := nodeLang(t, "javascript")
	const meta = `{"classname":"LRUCache","constructor":{"params":[{"name":"capacity","type":"integer"}]},` +
		`"methods":[{"name":"get","params":[{"name":"key","type":"integer"}],"return":{"type":"integer"}},` +
		`{"name":"put","params":[{"name":"key","type":"integer"},{"name":"value","type":"integer"}],"return":{"type":"void"}}],` +
		`"return":{"type":"integer"}}`

	dir := generateJSDir(t, l, lang, "lru-cache", meta, `
var LRUCache = function(capacity) { this.cap = capacity; this.m = new Map(); };
LRUCache.prototype.get = function(key) {
    if (!this.m.has(key)) return -1;
    const v = this.m.get(key);
    this.m.delete(key); this.m.set(key, v);
    return v;
};
LRUCache.prototype.put = function(key, value) {
    if (this.m.has(key)) this.m.delete(key);
    this.m.set(key, value);
    if (this.m.size > this.cap) this.m.delete(this.m.keys().next().value);
};
`)

	res := runJSCases(t, l, lang, dir, "lru-cache", []TestCase{{
		Input:    `["LRUCache","put","put","get","put","get"]` + "\n" + `[[2],[1,1],[2,2],[1],[3,3],[2]]`,
		Expected: "[null,null,null,1,null,-1]",
	}})
	if !res.Passed() {
		t.Errorf("design problem failed: %+v", res.Cases)
	}
}

// TestJavaScriptWrongAnswerIsData — a failing case is a result, not an error. The error
// return is reserved for the run not happening at all.
func TestJavaScriptWrongAnswerIsData(t *testing.T) {
	l, lang := nodeLang(t, "javascript")
	dir := generateJSDir(t, l, lang, "two-sum", twoSumMeta,
		"var twoSum = function(nums, target) { return [9, 9]; };\n")

	res := runJSCases(t, l, lang, dir, "two-sum",
		[]TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}})
	if res.Passed() {
		t.Error("a wrong answer reported as passing")
	}
	if res.Cases[0].Err != nil {
		t.Errorf("a wrong answer was reported as an error: %v", res.Cases[0].Err)
	}
}
