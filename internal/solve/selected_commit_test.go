package solve

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/workspace"
)

func TestCommitPathsUseSelectedDirectoryAndFilename(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "9999-selected-answer")
	if err := os.MkdirAll(filepath.Join(dir, "attempts"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("attempts", "my answer.py")
	for _, name := range []string{"README.md", "testcases.txt", file, "solution.py"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := commitPaths(ws, config.Git{}, Solved{ID: 1, Slug: "selected-answer", Dir: dir, Filename: file})
	want := []string{filepath.Join(dir, "README.md"), filepath.Join(dir, "testcases.txt"), filepath.Join(dir, file)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commit selection = %v, want %v", got, want)
	}
}
