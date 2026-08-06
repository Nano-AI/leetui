package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testWorkspace(t *testing.T) Workspace {
	t.Helper()
	w, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}
	return w
}

func sampleProblem() Problem {
	return Problem{
		ID: 146, Slug: "lru-cache", Title: "LRU Cache",
		Statement:  "Design a data structure that follows the constraints of an LRU cache.\n",
		Difficulty: "Medium",
		URL:        "https://leetcode.com/problems/lru-cache/",
	}
}

func TestFolderName(t *testing.T) {
	cases := []struct {
		id         int
		slug, want string
	}{
		{1, "two-sum", "0001-two-sum"},
		{146, "lru-cache", "0146-lru-cache"},
		{1650, "lowest-common-ancestor-iii", "1650-lowest-common-ancestor-iii"},
		{0, "lcp-01-guess-numbers", "0000-lcp-01-guess-numbers"},
		// The slug is remote data, so it is sanitised rather than trusted.
		{7, "Reverse Integer!", "0007-reverse-integer"},
		{8, "../../etc/passwd", "0008-etc-passwd"},
		{9, "", "0009-problem"},
	}
	for _, tc := range cases {
		if got := FolderName(tc.id, tc.slug); got != tc.want {
			t.Errorf("FolderName(%d, %q) = %q, want %q", tc.id, tc.slug, got, tc.want)
		}
	}
}

// TestFolderNameCannotEscape is the security check behind the sanitiser: a hostile slug
// must never produce a path outside the workspace root.
func TestFolderNameCannotEscape(t *testing.T) {
	w := testWorkspace(t)
	for _, slug := range []string{"../evil", "../../etc/passwd", "a/b/c", "..", "./."} {
		dir := w.Dir(1, slug)
		if !strings.HasPrefix(filepath.Clean(dir), filepath.Clean(w.Root)+string(filepath.Separator)) {
			t.Errorf("slug %q escaped the workspace: %s", slug, dir)
		}
	}
}

func TestCreateLaysOutFolder(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()

	dir, err := w.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if filepath.Base(dir) != "0146-lru-cache" {
		t.Errorf("folder = %s", filepath.Base(dir))
	}

	readme, err := w.ReadFile(p.ID, p.Slug, ReadmeFile)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	for _, want := range []string{"# 146. LRU Cache", "**Medium**", "leetcode.com", "LRU cache"} {
		if !strings.Contains(readme, want) {
			t.Errorf("README missing %q:\n%s", want, readme)
		}
	}
	if _, err := w.ReadFile(p.ID, p.Slug, NotesFile); err != nil {
		t.Errorf("notes not created: %v", err)
	}
}

// TestCreateRefreshesReadmeButKeepsNotes covers the asymmetry that matters: the README
// is derived and may be regenerated; notes are the user's and must survive.
func TestCreateRefreshesReadmeButKeepsNotes(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()
	if _, err := w.Create(p); err != nil {
		t.Fatal(err)
	}

	mine := "# my notes\n\nUse a hashmap plus a doubly linked list.\n"
	if err := os.WriteFile(w.Path(p.ID, p.Slug, NotesFile), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	p.Statement = "Updated statement from a later sync.\n"
	if _, err := w.Create(p); err != nil {
		t.Fatal(err)
	}

	notes, _ := w.ReadFile(p.ID, p.Slug, NotesFile)
	if notes != mine {
		t.Errorf("notes.md was overwritten by a re-sync:\n%s", notes)
	}
	readme, _ := w.ReadFile(p.ID, p.Slug, ReadmeFile)
	if !strings.Contains(readme, "Updated statement") {
		t.Error("README was not refreshed")
	}
}

// TestWriteSolutionNeverOverwrites is the guarantee this package exists to make.
func TestWriteSolutionNeverOverwrites(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()

	path, created, err := w.WriteSolution(p.ID, p.Slug, "solution.go", "func stub() {}")
	if err != nil || !created {
		t.Fatalf("first write: created=%v err=%v", created, err)
	}

	work := "func solved() { /* three hours of my life */ }"
	if err := os.WriteFile(path, []byte(work), 0o644); err != nil {
		t.Fatal(err)
	}

	_, created, err = w.WriteSolution(p.ID, p.Slug, "solution.go", "func stub() {}")
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if created {
		t.Error("reported creating a file that already existed")
	}
	got, _ := w.ReadFile(p.ID, p.Slug, "solution.go")
	if got != work {
		t.Fatalf("existing solution was overwritten:\n%s", got)
	}
}

func TestMultipleLanguagesCoexist(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()
	for _, f := range []string{"solution.go", "solution.py", "solution.cpp"} {
		if _, _, err := w.WriteSolution(p.ID, p.Slug, f, "stub"); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.Solutions(p.ID, p.Slug); len(got) != 3 {
		t.Errorf("Solutions() = %v, want 3", got)
	}
}

func TestTestcasesAreNotClobbered(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()

	if _, err := w.WriteTestcases(p.ID, p.Slug, "[2,7,11,15]\n9\n"); err != nil {
		t.Fatal(err)
	}
	mine := "[2,7,11,15]\n9\n[1,2]\n3\n"
	if err := os.WriteFile(w.Path(p.ID, p.Slug, TestcasesFile), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteTestcases(p.ID, p.Slug, "[2,7,11,15]\n9\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ReadFile(p.ID, p.Slug, TestcasesFile)
	if got != mine {
		t.Errorf("hand-added test cases were lost:\n%s", got)
	}
}

func TestModTimeDetectsExternalSave(t *testing.T) {
	w := testWorkspace(t)
	p := sampleProblem()

	// A file that does not exist reports the zero time, not an error.
	zero, err := w.ModTime(p.ID, p.Slug, "solution.go")
	if err != nil || !zero.IsZero() {
		t.Fatalf("missing file: %v, %v", zero, err)
	}

	path, _, err := w.WriteSolution(p.ID, p.Slug, "solution.go", "stub")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := w.ModTime(p.ID, p.Slug, "solution.go")

	if err := os.Chtimes(path, before.Add(time.Second), before.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	after, _ := w.ModTime(p.ID, p.Slug, "solution.go")
	if !after.After(before) {
		t.Error("a later write did not move the modification time")
	}
}

func TestNewExpandsTilde(t *testing.T) {
	w, err := New("~/leetcode")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(w.Root, "~") {
		t.Errorf("tilde was not expanded: %s", w.Root)
	}
	if !filepath.IsAbs(w.Root) {
		t.Errorf("root is not absolute: %s", w.Root)
	}
}
