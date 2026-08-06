package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/workspace"
)

func TestNewSolutionFileIsScaffolded(t *testing.T) {
	m := boot(t, true, 120, 32)
	ws := prepared(t, m, t.TempDir())

	got := mustRead(t, ws, "solution.py")

	for _, want := range []string{
		"# 1. Two Sum · Easy", // says what it is when opened alone
		"https://leetcode.com/problems/two-sum/",
		"from typing import", // List[int] resolves
		"# @leetui code=start",
		"class Solution:",
		"# @leetui code=end",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("scaffolded file is missing %q:\n%s", want, got)
		}
	}
}

// TestExistingSolutionIsWrappedNotReplaced is the guarantee that makes upgrading a file
// the user may have edited defensible: their code goes inside the markers byte for byte.
func TestExistingSolutionIsWrappedNotReplaced(t *testing.T) {
	m := boot(t, true, 120, 32)

	dir := t.TempDir()
	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	// An edited file from before scaffolding existed: no header, no imports, no markers.
	mine := "class Solution:\n    def twoSum(self, nums, target):\n        return [0, 1]  # my work\n"
	if _, _, err := ws.WriteSolution(1, "two-sum", "solution.py", mine); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ws = prepared(t, m, dir)
	got := mustRead(t, ws, "solution.py")

	if !strings.Contains(got, "# @leetui code=start") {
		t.Errorf("an existing file was not given scaffolding:\n%s", got)
	}
	if !strings.Contains(got, "return [0, 1]  # my work") {
		t.Fatalf("the user's code did not survive the upgrade:\n%s", got)
	}
	// And what is submitted is still exactly what they wrote.
	lang, _ := runner.Lookup("python3")
	if code := runner.ExtractCode(lang, got); strings.TrimSpace(code) != strings.TrimSpace(mine) {
		t.Errorf("submitted code changed:\ngot:\n%s\nwant:\n%s", code, mine)
	}
}

// TestWrappingIsIdempotent: prepare runs on every edit, run, and submit, so a file that
// already has markers must come out untouched.
func TestWrappingIsIdempotent(t *testing.T) {
	m := boot(t, true, 120, 32)
	dir := t.TempDir()

	ws := prepared(t, m, dir)
	first := mustRead(t, ws, "solution.py")

	ws = prepared(t, m, dir)
	if second := mustRead(t, ws, "solution.py"); second != first {
		t.Errorf("a second prepare changed the file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestGoPackageClauseMovesOutOfTheMarkedRegion: LeetCode wraps a Go submission in its own
// package, so a second clause is a compile error there. Wrapping has to lift it into the
// scaffolding rather than leave it in the submitted region.
func TestGoPackageClauseMovesOutOfTheMarkedRegion(t *testing.T) {
	lang, _ := runner.Lookup("golang")
	meta, err := runner.ParseMeta(twoSumMeta)
	if err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	s := runner.Scaffold{Lang: lang, Meta: meta, ID: 1, Title: "Two Sum"}

	old := "package main\n\nfunc twoSum(nums []int, target int) []int {\n\treturn nil\n}\n"
	wrapped, changed := s.Wrap(old)
	if !changed {
		t.Fatal("an unmarked Go file was not wrapped")
	}

	code := runner.ExtractCode(lang, wrapped)
	if strings.Contains(code, "package main") {
		t.Errorf("the package clause is still inside the marked region:\n%s", code)
	}
	if !strings.Contains(wrapped, "package main") {
		t.Errorf("the package clause was dropped instead of moved:\n%s", wrapped)
	}
	if !strings.Contains(code, "func twoSum") {
		t.Errorf("the function did not survive:\n%s", code)
	}
}
