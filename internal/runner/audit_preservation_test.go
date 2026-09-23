package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateIncludeLeavesUserCodeUntouched(t *testing.T) {
	for _, body := range []string{
		"// @leetui code=start\n#include \"leetui_driver.h\"\n// @leetui code=end\n",
		"#include \"leetui_driver.h\"\nclass Solution {};\n",
		"// example: #include \"leetui_driver.h\"\n// @leetui code=start\nclass Solution {};\n",
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "solution.cpp")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		header := filepath.Join(dir, "leetui_driver.h")
		if err := os.WriteFile(header, []byte("user header"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := migrateInclude(dir, "leetui_driver.h"); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != body {
			t.Errorf("user code changed: %q, %v", got, err)
		}
		if _, err := os.Stat(header); err != nil {
			t.Errorf("user header removed: %v", err)
		}
	}
}

func TestOrderedRowsInUnorderedAnswers(t *testing.T) {
	for _, slug := range []string{"permutations", "permutations-ii", "palindrome-partitioning"} {
		rule := RuleFor(slug)
		if Compare(`[[1,2],[1,2]]`, `[[1,2],[2,1]]`, rule) {
			t.Errorf("%s accepted duplicate rows in place of distinct sequences", slug)
		}
		if !Compare(`[[2,1],[1,2]]`, `[[1,2],[2,1]]`, rule) {
			t.Errorf("%s rejected reordered outer list", slug)
		}
	}
}

func TestGoModPreservesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "user-module")
	if err := os.Symlink(target, filepath.Join(dir, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if err := writeGoMod(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("go.mod generation wrote through user's dangling symlink: %v", err)
	}
}
