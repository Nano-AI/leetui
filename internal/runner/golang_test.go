package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func goLang(t *testing.T) (*Local, Lang) {
	t.Helper()
	l := NewLocal()
	lang, _ := Lookup("golang")
	if !l.Supports(lang) {
		t.Skip("go toolchain missing")
	}
	return l, lang
}

func genGo(t *testing.T, l *Local, lang Lang, slug, meta, solution string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "solution.go"), []byte(lang.SolutionFile(solution)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.Generate(context.Background(), Problem{Slug: slug, MetaData: meta}, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Generate must supply go.mod itself; `go build` refuses to run outside a module.
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("Generate did not write go.mod: %v", err)
	}
	return dir
}

func TestGoTwoSum(t *testing.T) {
	l, lang := goLang(t)
	dir := genGo(t, l, lang, "two-sum", twoSumMeta, `
func twoSum(nums []int, target int) []int {
	seen := map[int]int{}
	for i, n := range nums {
		if j, ok := seen[target-n]; ok {
			return []int{j, i}
		}
		seen[n] = i
	}
	return nil
}
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}}, RuleFor("two-sum"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CompileErr != "" {
		t.Fatalf("compile error:\n%s", res.CompileErr)
	}
	if !res.Passed() {
		t.Errorf("actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestGoLinkedListAndTree(t *testing.T) {
	l, lang := goLang(t)

	dir := genGo(t, l, lang, "reverse-linked-list",
		`{"name":"reverseList","params":[{"name":"head","type":"ListNode"}],"return":{"type":"ListNode"}}`, `
func reverseList(head *ListNode) *ListNode {
	var prev *ListNode
	for head != nil {
		head.Next, prev, head = prev, head, head.Next
	}
	return prev
}
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[1,2,3]", Expected: "[3,2,1]"}}, DefaultRule())
	if err != nil || res.CompileErr != "" {
		t.Fatalf("run: %v %s", err, res.CompileErr)
	}
	if !res.Passed() {
		t.Errorf("list: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}

	dir = genGo(t, l, lang, "maximum-depth-of-binary-tree",
		`{"name":"maxDepth","params":[{"name":"root","type":"TreeNode"}],"return":{"type":"integer"}}`, `
func maxDepth(root *TreeNode) int {
	if root == nil {
		return 0
	}
	a, b := maxDepth(root.Left), maxDepth(root.Right)
	if a > b {
		return a + 1
	}
	return b + 1
}
`)
	res, err = l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[3,9,20,null,null,15,7]", Expected: "3"}}, DefaultRule())
	if err != nil || res.CompileErr != "" {
		t.Fatalf("run: %v %s", err, res.CompileErr)
	}
	if !res.Passed() {
		t.Errorf("tree: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestGoCompileErrorIsReported(t *testing.T) {
	l, lang := goLang(t)
	dir := genGo(t, l, lang, "two-sum", twoSumMeta, `
func twoSum(nums []int, target int) []int {
	this is not go
}
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[2,7]\n9", Expected: "[0,1]"}}, DefaultRule())
	if err != nil {
		t.Fatalf("run returned an error instead of a compile result: %v", err)
	}
	if res.CompileErr == "" {
		t.Fatal("compile error not captured")
	}
	if res.Passed() {
		t.Error("a non-compiling solution reported as passing")
	}
}

func TestGoInPlace(t *testing.T) {
	l, lang := goLang(t)
	dir := genGo(t, l, lang, "remove-duplicates-from-sorted-array",
		`{"name":"removeDuplicates","params":[{"name":"nums","type":"integer[]"}],"return":{"type":"integer"}}`, `
func removeDuplicates(nums []int) int {
	k := 0
	for _, n := range nums {
		if k == 0 || nums[k-1] != n {
			nums[k] = n
			k++
		}
	}
	return k
}
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[1,1,2]", Expected: "[1,2]"}},
		RuleFor("remove-duplicates-from-sorted-array"))
	if err != nil || res.CompileErr != "" {
		t.Fatalf("run: %v %s", err, res.CompileErr)
	}
	if !res.Passed() {
		t.Errorf("in-place: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}
