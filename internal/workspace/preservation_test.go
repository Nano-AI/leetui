package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNeverOverwriteDoesNotFollowDanglingSymlinks(t *testing.T) {
	w, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir, err := w.Create(Problem{ID: 1, Slug: "two-sum"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"solution.py", TestcasesFile} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "absent")
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
			if name == TestcasesFile {
				_, err = w.WriteTestcases(1, "two-sum", "seed")
			} else {
				var created bool
				_, created, err = w.WriteSolution(1, "two-sum", name, "seed")
				if created {
					t.Error("existing symlink reported as newly created")
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Errorf("write followed symlink: %v", err)
			}
		})
	}
}
