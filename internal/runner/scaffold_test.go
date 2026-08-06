package runner

import (
	"strings"
	"testing"
)

const listMeta = `{"name":"reverseList","params":[{"name":"head","type":"ListNode"}],
	"return":{"type":"ListNode"}}`

func scaffoldFor(t *testing.T, slug, meta string) Scaffold {
	t.Helper()
	lang, ok := Lookup(slug)
	if !ok {
		t.Fatalf("unknown language %q", slug)
	}
	m, err := ParseMeta(meta)
	if err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	return Scaffold{Lang: lang, Meta: m, ID: 1, Title: "Two Sum", Difficulty: "Easy",
		URL: "https://leetcode.com/problems/two-sum/"}
}

// TestScaffoldGivesTheEditorWhatItNeeds is the point of the whole file: a LeetCode
// snippet written to disk raw is a buffer full of unresolved symbols, because the judge
// supplies the imports itself.
func TestScaffoldGivesTheEditorWhatItNeeds(t *testing.T) {
	for _, tc := range []struct{ slug, meta, want string }{
		// One include carries the std headers, `using namespace std`, and the node types,
		// so `vector<int>` resolves.
		{"cpp", twoSumMeta, `#include "leetui_driver.h"`},
		// The driver shares this package, which is what puts ListNode in scope.
		{"golang", twoSumMeta, "package main"},
		// LeetCode's annotations say List[int] and Optional[ListNode].
		{"python3", twoSumMeta, "from typing import"},
		{"java", twoSumMeta, "import java.util.*;"},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			got := scaffoldFor(t, tc.slug, tc.meta).File("// body\n")
			if !strings.Contains(got, tc.want) {
				t.Errorf("scaffold is missing %q:\n%s", tc.want, got)
			}
		})
	}
}

// TestPythonImportsNodesOnlyWhenUsed keeps the file free of imports a linter would flag,
// and — more importantly — imports them from the DRIVER rather than declaring them. The
// driver serializes with isinstance against its own classes, so a solution returning a
// locally-declared ListNode would fail to serialize and read as a wrong answer.
func TestPythonImportsNodesOnlyWhenUsed(t *testing.T) {
	withNodes := scaffoldFor(t, "python3", listMeta).File("pass\n")
	if !strings.Contains(withNodes, "from _leetui_driver import ListNode") {
		t.Errorf("a ListNode problem did not import it from the driver:\n%s", withNodes)
	}

	plain := scaffoldFor(t, "python3", twoSumMeta).File("pass\n")
	if strings.Contains(plain, "_leetui_driver") {
		t.Errorf("imported node types into a problem that has none:\n%s", plain)
	}
}

func TestScaffoldUsesTheRightCommentToken(t *testing.T) {
	py := scaffoldFor(t, "python3", twoSumMeta).File("pass\n")
	if strings.Contains(py, "// ") {
		t.Errorf("python scaffold used C-style comments:\n%s", py)
	}
	if !strings.Contains(py, "# @leetui code=start") {
		t.Errorf("python marker is not a python comment:\n%s", py)
	}
}

// TestScaffoldRoundTripsThroughExtract is the invariant that keeps the two halves
// honest: whatever is wrapped comes back out byte-for-byte, or the judge sees something
// the user never wrote.
func TestScaffoldRoundTripsThroughExtract(t *testing.T) {
	body := "class Solution {\npublic:\n    vector<int> twoSum(vector<int>& nums, int target) {\n        return {};\n    }\n};"

	for _, slug := range []string{"cpp", "golang", "python3", "java", "rust"} {
		t.Run(slug, func(t *testing.T) {
			s := scaffoldFor(t, slug, twoSumMeta)
			got := ExtractCode(s.Lang, s.File(body))
			if strings.TrimSpace(got) != strings.TrimSpace(body) {
				t.Errorf("round trip changed the code:\ngot:\n%s\nwant:\n%s", got, body)
			}
		})
	}
}
