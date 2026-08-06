package runner

import (
	"strings"
	"testing"
)

func lang(t *testing.T, slug string) Lang {
	t.Helper()
	l, ok := Lookup(slug)
	if !ok {
		t.Fatalf("unknown language %q", slug)
	}
	return l
}

func TestExtractTakesOnlyTheMarkedRegion(t *testing.T) {
	file := `// 1. Two Sum · Easy
// https://leetcode.com/problems/two-sum/

#include "leetui_driver.h"

// @leetui code=start
class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) { return {}; }
};
// @leetui code=end
`
	got := ExtractCode(lang(t, "cpp"), file)
	if strings.Contains(got, "leetui_driver.h") {
		t.Errorf("the driver include leaked into the submission:\n%s", got)
	}
	if strings.Contains(got, "@leetui") {
		t.Errorf("a marker leaked into the submission:\n%s", got)
	}
	if !strings.HasPrefix(got, "class Solution {") {
		t.Errorf("submission does not start at the marked code:\n%s", got)
	}
}

// TestExtractReadsVscodeMarkers means a workspace built with the vscode-leetcode
// extension submits correctly here without being rewritten.
func TestExtractReadsVscodeMarkers(t *testing.T) {
	file := `/*
 * @lc app=leetcode id=1 lang=golang
 */
package main

// @lc code=start
func twoSum(nums []int, target int) []int { return nil }
// @lc code=end
`
	got := ExtractCode(lang(t, "golang"), file)
	if strings.Contains(got, "package main") {
		t.Errorf("package clause survived:\n%s", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(got), "func twoSum") {
		t.Errorf("did not extract the marked function:\n%s", got)
	}
}

// TestExtractStripsGoPackageWithoutMarkers covers files written before the markers
// existed. LeetCode wraps a Go submission in a package of its own, so a second package
// clause is a compile error on the judge — the one piece of scaffolding that cannot
// simply be sent along.
func TestExtractStripsGoPackageWithoutMarkers(t *testing.T) {
	file := "package main\n\nfunc twoSum(nums []int, target int) []int {\n\treturn nil\n}\n"

	got := ExtractCode(lang(t, "golang"), file)
	if strings.Contains(got, "package main") {
		t.Errorf("package clause survived an unmarked file:\n%s", got)
	}
	if !strings.Contains(got, "func twoSum") {
		t.Errorf("stripping took the function with it:\n%s", got)
	}
}

// TestExtractLeavesOtherLanguagesAlone: LeetCode accepts C++ includes and Python imports,
// so an unmarked file in those languages is sent as it is rather than guessed at.
func TestExtractLeavesOtherLanguagesAlone(t *testing.T) {
	file := "#include <vector>\nusing namespace std;\n\nclass Solution {};\n"
	if got := ExtractCode(lang(t, "cpp"), file); got != file {
		t.Errorf("an unmarked C++ file was modified:\ngot:\n%s\nwant:\n%s", got, file)
	}
}

// TestExtractSurvivesAMissingEndMarker: a half-deleted marker pair is far more likely to
// be a stray edit than a request to submit nothing.
func TestExtractSurvivesAMissingEndMarker(t *testing.T) {
	file := "package main\n\n// @leetui code=start\nfunc twoSum() {}\n"

	got := ExtractCode(lang(t, "golang"), file)
	if !strings.Contains(got, "func twoSum") {
		t.Errorf("lost the code to a missing end marker:\n%q", got)
	}
	if strings.Contains(got, "package main") {
		t.Errorf("kept the scaffolding despite a start marker:\n%q", got)
	}
}

func TestLegacyStarterMatchesWhatOldVersionsWrote(t *testing.T) {
	snippet := "func twoSum(nums []int, target int) []int {\n\n}"

	if got := LegacyStarter(lang(t, "golang"), snippet); got != "package main\n\n"+snippet {
		t.Errorf("go legacy starter is %q", got)
	}
	if got := LegacyStarter(lang(t, "python3"), snippet); got != snippet {
		t.Errorf("python legacy starter added something: %q", got)
	}
}
