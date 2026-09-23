package solve

import (
	"testing"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
)

func TestPrepareDesignCasesKeepOperationsAndArgumentsTogether(t *testing.T) {
	d := &store.Detail{}
	d.NumericID, d.Slug, d.Title = 155, "min-stack", "Min Stack"
	d.MetaData = `{"classname":"MinStack","constructor":{"params":[]},"methods":[{"name":"top","params":[],"return":{"type":"integer"}}]}`
	d.Snippets = map[string]string{"python3": "class MinStack: pass"}
	d.ExampleTestcases = "[\"MinStack\",\"top\"]\n[[],[]]"
	d.Content = "<pre>Output: [null,1]</pre>"
	lang, _ := runner.Lookup("python3")
	out, err := Prepare(t.TempDir(), d, lang)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := Cases(out.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Input != d.ExampleTestcases || cases[0].Expected != "[null,1]" {
		t.Fatalf("design input split into invalid cases: %+v", cases)
	}
}
