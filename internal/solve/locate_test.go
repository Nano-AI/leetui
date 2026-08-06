package solve

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocateFromSlug(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"two-sum", "two-sum"},
		{"  two-sum  ", "two-sum"},
		// A folder name carries the id; the API only knows the slug.
		{"0001-two-sum", "two-sum"},
		{"146-lru-cache", "lru-cache"},
		// A slug with digits in it must survive.
		{"3sum", "3sum"},
	} {
		got, err := Locate(tc.in)
		if err != nil {
			t.Errorf("Locate(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Locate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestLocateFromFilePath is the one that makes `:!leetui run %` work from inside nvim.
func TestLocateFromFilePath(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "0001-two-sum")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "solution.py")
	if err := os.WriteFile(file, []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, in := range []string{file, dir} {
		got, err := Locate(in)
		if err != nil {
			t.Errorf("Locate(%q): %v", in, err)
			continue
		}
		if got != "two-sum" {
			t.Errorf("Locate(%q) = %q, want two-sum", in, got)
		}
	}
}

// TestLocateWalksUp: a Go problem folder holds generated files, and a user may be editing
// one of them when they ask for a run.
func TestLocateWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "0146-lru-cache", "sub", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(nested)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got != "lru-cache" {
		t.Errorf("Locate(nested) = %q, want lru-cache", got)
	}
}

func TestLocateFromWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "0042-trapping-rain-water")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := Locate("")
	if err != nil {
		t.Fatalf("Locate(\"\"): %v", err)
	}
	if got != "trapping-rain-water" {
		t.Errorf("Locate from cwd = %q, want trapping-rain-water", got)
	}
}

func TestLocateOutsideAProblemFolder(t *testing.T) {
	t.Chdir(t.TempDir())

	if _, err := Locate(""); !errors.Is(err, ErrNoProblem) {
		t.Errorf("Locate outside a problem folder returned %v, want ErrNoProblem", err)
	}
}

// TestLocatePrefersAnExistingPath: a slug that happens to match a directory in the
// working directory must not be misread as a path to somewhere else.
func TestLocatePrefersAnExistingPath(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, "0001-two-sum"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Locate("0001-two-sum")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got != "two-sum" {
		t.Errorf("Locate = %q, want two-sum", got)
	}
}
