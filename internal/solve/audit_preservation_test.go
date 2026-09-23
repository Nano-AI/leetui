package solve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

func TestSeedCasesPreservesCustomUncheckedInputs(t *testing.T) {
	for _, existing := range []string{"99\noutput:\n\n", "", "1\noutput:\n\n99\noutput:\n\n"} {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		d := &store.Detail{}
		d.NumericID, d.Slug = 1, "test"
		path, err := ws.WriteTestcases(1, "test", existing)
		if err != nil {
			t.Fatal(err)
		}
		seedCases(ws, d, []runner.TestCase{{Input: "1", Expected: "2"}})
		got, err := os.ReadFile(path)
		if err != nil || string(got) != existing {
			t.Errorf("%s: lost custom cases %q: got %q, %v", filepath.Base(path), existing, got, err)
		}
	}
}

func TestSeedCasesRepairsOnlyOriginalBlankExamples(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := &store.Detail{}
	d.NumericID, d.Slug = 1, "test"
	path, err := ws.WriteTestcases(1, "test", runner.FormatCases([]runner.TestCase{{Input: "1"}}))
	if err != nil {
		t.Fatal(err)
	}
	seedCases(ws, d, []runner.TestCase{{Input: "1", Expected: "2"}})
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "1\noutput:\n2\n" {
		t.Fatalf("legacy repair = %q, %v", got, err)
	}
}
