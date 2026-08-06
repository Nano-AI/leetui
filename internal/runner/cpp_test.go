package runner

import (
	"context"
	"os"
	"path/filepath"
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
