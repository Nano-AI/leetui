package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	if err := os.WriteFile(filepath.Join(dir, "solution.go"), []byte(scaffolded(t, lang, meta, solution)), 0o644); err != nil {
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

// TestGoInPlaceVoidReturn is the Go half of the same regression: `n := moveZeroes(a0)`
// against a function that returns nothing is "moveZeroes(a0) (no value) used as value".
func TestGoInPlaceVoidReturn(t *testing.T) {
	l, lang := goLang(t)
	dir := genGo(t, l, lang, "move-zeroes", moveZeroesMeta, `
func moveZeroes(nums []int) {
	k := 0
	for _, n := range nums {
		if n != 0 {
			nums[k] = n
			k++
		}
	}
	for ; k < len(nums); k++ {
		nums[k] = 0
	}
}
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{
			{Input: "[0,1,0,3,12]", Expected: "[1,3,12,0,0]"},
			{Input: "[0]", Expected: "[0]"},
		}, RuleFor("move-zeroes"))
	if err != nil || res.CompileErr != "" {
		t.Fatalf("run: %v %s", err, res.CompileErr)
	}
	if !res.Passed() {
		t.Errorf("void in-place: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

// TestGoInPlaceCallShape is the toolchain-free companion to TestGoInPlaceVoidReturn, for
// the same reason the C++ one exists: goLang skips when the toolchain is missing.
func TestGoInPlaceCallShape(t *testing.T) {
	lang, _ := Lookup("golang")
	cases := []struct {
		name, slug, meta string
		want, notWant    []string
	}{
		{
			name: "void answers with the whole argument",
			slug: "move-zeroes",
			meta: moveZeroesMeta,
			want: []string{"\tmoveZeroes(a0)\n", "serialize(a0)"},
			// prefix stays declared in every generated body, so match the call, not
			// the word.
			notWant: []string{"n := moveZeroes", "prefix(a0"},
		},
		{
			name:    "a reported length still trims",
			slug:    "remove-duplicates-from-sorted-array",
			meta:    removeDuplicatesMeta,
			want:    []string{"n := removeDuplicates(a0)", "serialize(prefix(a0, n))"},
			notWant: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := NewLocal().Generate(context.Background(),
				Problem{Slug: tc.slug, MetaData: tc.meta}, lang, dir); err != nil {
				t.Fatalf("generate: %v", err)
			}
			body, err := os.ReadFile(filepath.Join(dir, goDriverFile))
			if err != nil {
				t.Fatal(err)
			}
			got := string(body)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("generated driver is missing %q:\n%s", w, got)
				}
			}
			for _, n := range tc.notWant {
				if strings.Contains(got, n) {
					t.Errorf("generated driver still contains %q:\n%s", n, got)
				}
			}
		})
	}
}
