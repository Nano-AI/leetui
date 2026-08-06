package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
)

// ready() needs a loaded detail, which boot alone does not give.
func openTwoSum(t *testing.T, dir string) Model {
	t.Helper()
	m := boot(t, true, 120, 30)
	m.cfg.Workspace = dir
	// Seed the store only. Laying out the folder here would mean the first create was
	// never the first, which is exactly what the wording branch turns on.
	seedDetail(t, m)

	m = loadDetail(t, m)
	if m.detail == nil {
		t.Fatal("detail did not load")
	}
	return m
}

// TestCreateMakesTheFileWithoutAnEditor is the whole point: leetui creates it, you open it
// wherever you already are (D-016).
func TestCreateMakesTheFileWithoutAnEditor(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	m = drive(t, m, key("f"))
	if m.picking != pickCreate {
		t.Fatalf("f did not open the create picker; picking = %v", m.picking)
	}

	m = drive(t, m, key("enter"))

	path := filepath.Join(dir, "0001-two-sum", "solution.py")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file was not created: %v", err)
	}
	// Scaffolded, not touched: it should be openable, not merely present (D-014).
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "@leetui code=start") {
		t.Errorf("the created file has no scaffolding:\n%s", body)
	}
}

// TestCreateToastsThePath: the path is the thing you are about to type into another
// window, so it has to be on screen and readable.
func TestCreateToastsThePath(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	m = drive(t, m, key("f"), key("enter"))

	if m.toast == nil {
		t.Fatal("creating a file raised no toast")
	}
	if !strings.Contains(m.toast.title, "Python3") {
		t.Errorf("the toast does not name the language: %q", m.toast.title)
	}
	if !strings.Contains(m.toast.body, "0001-two-sum") {
		t.Errorf("the toast does not carry the path: %q", m.toast.body)
	}

	out := stripANSI(m.View())
	if !strings.Contains(out, "solution.py") {
		t.Errorf("the path is not on screen:\n%s", out)
	}
}

// TestCreateSaysReadyForAFileThatExisted: "Created" over a file you have been working in
// for an hour would be alarming, and it is the same keypress.
func TestCreateSaysReadyForAFileThatExisted(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	m = drive(t, m, key("f"), key("enter"))
	if !strings.Contains(strings.ToLower(m.toast.title), "created") {
		t.Fatalf("first create said %q", m.toast.title)
	}

	m = drive(t, m, key("f"), key("enter"))
	if strings.Contains(strings.ToLower(m.toast.title), "created") {
		t.Errorf("a second create still claimed to have created it: %q", m.toast.title)
	}
}

// TestCreateRemembersTheLanguage: whatever you were writing last time is a better guess
// than a default chosen once during setup.
func TestCreateRemembersTheLanguage(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	cpp, ok := runner.Lookup("cpp")
	if !ok {
		t.Fatal("cpp is not a known language")
	}
	m2, _ := m.createFile(cpp)
	after := m2.(Model)

	if after.lang.Slug != "cpp" {
		t.Errorf("language is %q after creating a C++ file", after.lang.Slug)
	}
	if _, err := os.Stat(filepath.Join(dir, "0001-two-sum", "solution.cpp")); err != nil {
		t.Errorf("solution.cpp was not created: %v", err)
	}
	// The picker should now open on the language just used.
	after.picking = pickCreate
	if idx := after.langIndex(); after.pickerLangs()[idx].Slug != "cpp" {
		t.Errorf("the picker would open on %q, want cpp", after.pickerLangs()[idx].Slug)
	}
}

// TestAnyKeyDismissesTheToast: it is covering content, so the next deliberate action
// clears it rather than making the user wait out the timer.
func TestAnyKeyDismissesTheToast(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	m = drive(t, m, key("f"), key("enter"))
	if m.toast == nil {
		t.Fatal("no toast to dismiss")
	}

	m = drive(t, m, key("j"))
	if m.toast != nil {
		t.Error("the toast survived a keypress")
	}
}

// TestCreateDoesNotArmTheWatcher: the scaffolding write is not a save, and firing a run
// on it would report failures for a file the user has not touched.
func TestCreateDoesNotArmTheWatcher(t *testing.T) {
	dir := t.TempDir()
	m := openTwoSum(t, dir)

	m = drive(t, m, key("f"), key("enter"))
	m = drive(t, m, watchTick{})

	if m.runResult != nil {
		t.Error("creating the file triggered a run")
	}
}
