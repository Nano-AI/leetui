package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// The shared half of a driver is byte-identical for every problem, so it lives once in
// globals/ rather than beside every solution — ~19KB per folder, burying the files you
// edit under files you never open.

func TestDriverGoesToGlobals(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "0001-two-sum")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeGlobalDriver(dir, "leetui_driver.h", "drivers/cpp/driver.h"); err != nil {
		t.Fatalf("write: %v", err)
	}

	shared := filepath.Join(root, workspace.GlobalsDir, "leetui_driver.h")
	if _, err := os.Stat(shared); err != nil {
		t.Errorf("driver is not in globals/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "leetui_driver.h")); err == nil {
		t.Error("a copy was still left in the problem folder")
	}
}

// TestMigrateRewritesAnOldInclude covers every folder that existed before the move.
// Doing nothing would greet each of them with "fatal error: 'leetui_driver.h' file not
// found" — the include is a line leetui wrote, above the marker, in the region both the
// docs and scaffold.go call scaffolding rather than code.
func TestMigrateRewritesAnOldInclude(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "0001-two-sum")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	const before = `// 1. Two Sum
#include "leetui_driver.h"

// @leetui code=start
class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) { return {}; }
};
// @leetui code=end
`
	solution := filepath.Join(dir, "solution.cpp")
	if err := os.WriteFile(solution, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	// The stale copy the old layout left beside it.
	if err := os.WriteFile(filepath.Join(dir, "leetui_driver.h"), []byte("// old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := migrateInclude(dir, "leetui_driver.h"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	body, err := os.ReadFile(solution)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	if !strings.Contains(got, `#include "../globals/leetui_driver.h"`) {
		t.Errorf("include was not rewritten:\n%s", got)
	}
	// Everything below the marker is the user's, and must be untouched.
	if !strings.Contains(got, "vector<int> twoSum(vector<int>& nums, int target) { return {}; }") {
		t.Errorf("the solution body was disturbed:\n%s", got)
	}
	// The stale copy has to go, or the old include would still resolve and the
	// migration would look optional until the day it was not.
	if _, err := os.Stat(filepath.Join(dir, "leetui_driver.h")); err == nil {
		t.Error("the stale per-folder driver survived")
	}
}

// Migrating twice must not corrupt an already-migrated file.
func TestMigrateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	const already = "#include \"../globals/leetui_driver.h\"\n// @leetui code=start\n"
	solution := filepath.Join(dir, "solution.cpp")
	if err := os.WriteFile(solution, []byte(already), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if err := migrateInclude(dir, "leetui_driver.h"); err != nil {
			t.Fatalf("migrate %d: %v", i, err)
		}
	}
	body, _ := os.ReadFile(solution)
	if string(body) != already {
		t.Errorf("an already-migrated file was changed:\n%s", body)
	}
}

// A folder with no solution yet is the ordinary case on a fresh pull.
func TestMigrateOnAMissingSolution(t *testing.T) {
	if err := migrateInclude(t.TempDir(), "leetui_driver.h"); err != nil {
		t.Errorf("migrate with no solution: %v", err)
	}
}
