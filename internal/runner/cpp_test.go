package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cppLang(t *testing.T) (*Local, Lang) {
	t.Helper()
	l := NewLocal()
	lang, _ := Lookup("cpp")
	if !l.Supports(lang) {
		t.Skip("no c++ compiler")
	}
	return l, lang
}

func genCpp(t *testing.T, l *Local, lang Lang, slug, meta, solution string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "solution.cpp"),
		[]byte(scaffolded(t, lang, meta, solution)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.Generate(context.Background(),
		Problem{Slug: slug, MetaData: meta}, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return dir
}

func runCppCase(t *testing.T, l *Local, lang Lang, dir, slug string, cases []TestCase) Result {
	t.Helper()
	res, err := l.Run(context.Background(), dir, lang, cases, RuleFor(slug))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CompileErr != "" {
		t.Fatalf("compile error:\n%s", res.CompileErr)
	}
	return res
}

func TestCppTwoSum(t *testing.T) {
	l, lang := cppLang(t)
	dir := genCpp(t, l, lang, "two-sum", twoSumMeta, `
class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) {
        unordered_map<int,int> seen;
        for (int i = 0; i < (int)nums.size(); i++) {
            auto it = seen.find(target - nums[i]);
            if (it != seen.end()) return {it->second, i};
            seen[nums[i]] = i;
        }
        return {};
    }
};
`)
	res := runCppCase(t, l, lang, dir, "two-sum",
		[]TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}})
	if !res.Passed() {
		t.Errorf("actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestCppStringsAndTree(t *testing.T) {
	l, lang := cppLang(t)

	dir := genCpp(t, l, lang, "longest-common-prefix",
		`{"name":"longestCommonPrefix","params":[{"name":"strs","type":"string[]"}],
		  "return":{"type":"string"}}`, `
class Solution {
public:
    string longestCommonPrefix(vector<string>& strs) {
        if (strs.empty()) return "";
        string p = strs[0];
        for (auto& s : strs)
            while (s.compare(0, p.size(), p) != 0) p = p.substr(0, p.size()-1);
        return p;
    }
};
`)
	res := runCppCase(t, l, lang, dir, "longest-common-prefix",
		[]TestCase{{Input: `["flower","flow","flight"]`, Expected: `"fl"`}})
	if !res.Passed() {
		t.Errorf("strings: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}

	dir = genCpp(t, l, lang, "maximum-depth-of-binary-tree",
		`{"name":"maxDepth","params":[{"name":"root","type":"TreeNode"}],
		  "return":{"type":"integer"}}`, `
class Solution {
public:
    int maxDepth(TreeNode* root) {
        if (!root) return 0;
        return 1 + max(maxDepth(root->left), maxDepth(root->right));
    }
};
`)
	res = runCppCase(t, l, lang, dir, "maximum-depth-of-binary-tree",
		[]TestCase{{Input: "[3,9,20,null,null,15,7]", Expected: "3"}})
	if !res.Passed() {
		t.Errorf("tree: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestCppLinkedListAndInPlace(t *testing.T) {
	l, lang := cppLang(t)

	dir := genCpp(t, l, lang, "reverse-linked-list",
		`{"name":"reverseList","params":[{"name":"head","type":"ListNode"}],
		  "return":{"type":"ListNode"}}`, `
class Solution {
public:
    ListNode* reverseList(ListNode* head) {
        ListNode* prev = nullptr;
        while (head) { ListNode* n = head->next; head->next = prev; prev = head; head = n; }
        return prev;
    }
};
`)
	res := runCppCase(t, l, lang, dir, "reverse-linked-list",
		[]TestCase{{Input: "[1,2,3]", Expected: "[3,2,1]"}})
	if !res.Passed() {
		t.Errorf("list: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}

	dir = genCpp(t, l, lang, "remove-duplicates-from-sorted-array",
		`{"name":"removeDuplicates","params":[{"name":"nums","type":"integer[]"}],
		  "return":{"type":"integer"}}`, `
class Solution {
public:
    int removeDuplicates(vector<int>& nums) {
        int k = 0;
        for (int n : nums) if (k == 0 || nums[k-1] != n) nums[k++] = n;
        return k;
    }
};
`)
	res = runCppCase(t, l, lang, dir, "remove-duplicates-from-sorted-array",
		[]TestCase{{Input: "[1,1,2]", Expected: "[1,2]"}})
	if !res.Passed() {
		t.Errorf("in-place: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

// A vector of nodes, which is how every "merge k" problem states its input.
//
// C++ had ListNode and TreeNode but not vectors of them, so merge-k-sorted-lists
// failed to generate at all: "no C++ type for metaData type \"ListNode[]\"". Go
// had carried the same four keys since it was written.
func TestCppNodeVectors(t *testing.T) {
	l, lang := cppLang(t)

	dir := genCpp(t, l, lang, "merge-k-sorted-lists",
		`{"name":"mergeKLists","params":[{"name":"lists","type":"ListNode[]"}],
		  "return":{"type":"ListNode"}}`, `
class Solution {
public:
    ListNode* mergeKLists(vector<ListNode*>& lists) {
        ListNode dummy, *tail = &dummy;
        while (true) {
            ListNode** best = nullptr;
            for (auto& l : lists) if (l && (!best || l->val < (*best)->val)) best = &l;
            if (!best) break;
            tail->next = *best; tail = *best; *best = (*best)->next;
        }
        tail->next = nullptr;
        return dummy.next;
    }
};
`)
	res := runCppCase(t, l, lang, dir, "merge-k-sorted-lists",
		[]TestCase{
			{Input: "[[1,4,5],[1,3,4],[2,6]]", Expected: "[1,1,2,3,4,4,5,6]"},
			{Input: "[]", Expected: "[]"},
			{Input: "[[]]", Expected: "[]"},
		})
	if !res.Passed() {
		for i, c := range res.Cases {
			t.Errorf("lists case %d: actual=%q err=%v", i+1, c.Actual, c.Err)
		}
	}

	// The tree side of the same gap, and it returns a vector of nodes rather
	// than taking one, so dump resolves through dump(TreeNode*) as well.
	dir = genCpp(t, l, lang, "forest-echo",
		`{"name":"echoForest","params":[{"name":"roots","type":"TreeNode[]"}],
		  "return":{"type":"TreeNode[]"}}`, `
class Solution {
public:
    vector<TreeNode*> echoForest(vector<TreeNode*>& roots) { return roots; }
};
`)
	res = runCppCase(t, l, lang, dir, "forest-echo",
		[]TestCase{{Input: "[[1,2,3],[4]]", Expected: "[[1,2,3],[4]]"}})
	if !res.Passed() {
		t.Errorf("forest: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestCppCompileErrorIsReported(t *testing.T) {
	l, lang := cppLang(t)
	dir := genCpp(t, l, lang, "two-sum", twoSumMeta, `
class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) {
        this is not c++
    }
};
`)
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[2,7]\n9", Expected: "[0,1]"}}, DefaultRule())
	if err != nil {
		t.Fatalf("run returned an error instead of a compile result: %v", err)
	}
	if res.CompileErr == "" {
		t.Fatal("compile error not captured")
	}
}

// TestCppMissingReturnIsACompileError guards a real failure: a half-written solution with
// no return statement is undefined behaviour, and it used to compile clean and die as
// "signal: bus error" because -w cancelled -Wreturn-type. Mid-solution is exactly when a
// straight answer matters most.
func TestCppMissingReturnIsACompileError(t *testing.T) {
	l, lang := cppLang(t)
	dir := genCpp(t, l, lang, "two-sum", twoSumMeta, `class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) {
        unordered_map<int, int> seen;
        for (int i = 0; i < (int)nums.size(); i++) {
            seen[nums[i]] = i;
        }
    }
};`)

	// Not runCppCase: that helper fails the test on any compile error, which is right
	// everywhere except here, where the compile error IS the expected outcome.
	res, err := l.Run(context.Background(), dir, lang,
		[]TestCase{{Input: "[2,7,11,15]\n9", Expected: "[0,1]"}}, RuleFor("two-sum"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if res.CompileErr == "" {
		t.Fatalf("a missing return compiled; cases = %+v", res.Cases)
	}
	if !strings.Contains(res.CompileErr, "return") {
		t.Errorf("the compile error does not mention the return:\n%s", res.CompileErr)
	}
}

// TestCppInPlaceVoidReturn guards a compile error, not a wrong answer. A void in-place
// solution used to generate `auto n = sol.moveZeroes(a0);` — a variable of type void —
// because the generator asked whether the answer was a mutated argument and never
// whether there was a return value to trim by. Nine of the twelve in-place overrides
// return void; the three that do not were the whole of the coverage.
func TestCppInPlaceVoidReturn(t *testing.T) {
	l, lang := cppLang(t)

	dir := genCpp(t, l, lang, "move-zeroes", moveZeroesMeta, `
class Solution {
public:
    void moveZeroes(vector<int>& nums) {
        int k = 0;
        for (int n : nums) if (n != 0) nums[k++] = n;
        while (k < (int)nums.size()) nums[k++] = 0;
    }
};
`)
	res := runCppCase(t, l, lang, dir, "move-zeroes", []TestCase{
		{Input: "[0,1,0,3,12]", Expected: "[1,3,12,0,0]"},
		{Input: "[0]", Expected: "[0]"},
	})
	if !res.Passed() {
		t.Errorf("void in-place: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

// TestCppVoidAnswerIsNotTrimmed pins the case a prefix trim would quietly corrupt. merge
// takes m and n alongside the arrays, so a generator that reached for "the length the
// solution reported" has two wrong numbers within easy reach; the whole of nums1 is the
// answer.
func TestCppVoidAnswerIsNotTrimmed(t *testing.T) {
	l, lang := cppLang(t)

	dir := genCpp(t, l, lang, "merge-sorted-array", mergeSortedMeta, `
class Solution {
public:
    void merge(vector<int>& nums1, int m, vector<int>& nums2, int n) {
        int i = m - 1, j = n - 1, k = m + n - 1;
        while (j >= 0) nums1[k--] = (i >= 0 && nums1[i] > nums2[j]) ? nums1[i--] : nums2[j--];
    }
};
`)
	res := runCppCase(t, l, lang, dir, "merge-sorted-array", []TestCase{
		{Input: "[1,2,3,0,0,0]\n3\n[2,5,6]\n3", Expected: "[1,2,2,3,5,6]"},
	})
	if !res.Passed() {
		t.Errorf("merge: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

// TestCppInPlaceCallShape pins both generated call shapes without a compiler. cppLang
// skips when c++ is absent, so on such a machine TestCppInPlaceVoidReturn is a green
// skip and the regression walks straight back in. Generate never shells out, so this
// test always runs.
func TestCppInPlaceCallShape(t *testing.T) {
	lang, _ := Lookup("cpp")
	cases := []struct {
		name, slug, meta string
		want, notWant    []string
	}{
		{
			name: "void answers with the whole argument",
			slug: "move-zeroes",
			meta: moveZeroesMeta,
			want: []string{"sol.moveZeroes(a0);", "leetui::dump(a0)"},
			// Both halves of the old shape: the void variable and the trim it fed.
			notWant: []string{"auto n = sol.moveZeroes", "leetui::prefix"},
		},
		{
			name:    "a reported length still trims",
			slug:    "remove-duplicates-from-sorted-array",
			meta:    removeDuplicatesMeta,
			want:    []string{"auto n = sol.removeDuplicates(a0);", "leetui::prefix(a0, (int)n)"},
			notWant: nil,
		},
		{
			// Not in the override table. It used to print "null", which no test case can
			// ever match; a void solution can only be answering through its argument.
			name: "an uncurated void problem answers with its argument",
			slug: "recover-binary-search-tree",
			meta: `{"name":"recoverTree","params":[{"name":"root","type":"TreeNode"}],
				"return":{"type":"void"}}`,
			want:    []string{"sol.recoverTree(a0);", "leetui::dump(a0)"},
			notWant: []string{`"null"`, "leetui::prefix"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := NewLocal().Generate(context.Background(),
				Problem{Slug: tc.slug, MetaData: tc.meta}, lang, dir); err != nil {
				t.Fatalf("generate: %v", err)
			}
			body, err := os.ReadFile(filepath.Join(dir, cppMainFile))
			if err != nil {
				t.Fatal(err)
			}
			got := string(body)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("generated main is missing %q:\n%s", w, got)
				}
			}
			for _, n := range tc.notWant {
				if strings.Contains(got, n) {
					t.Errorf("generated main still contains %q:\n%s", n, got)
				}
			}
		})
	}
}
