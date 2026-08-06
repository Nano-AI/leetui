package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const lruMeta = `{"classname":"LRUCache",
	"constructor":{"params":[{"name":"capacity","type":"integer"}]},
	"methods":[
		{"name":"get","params":[{"name":"key","type":"integer"}],"return":{"type":"integer"}},
		{"name":"put","params":[{"name":"key","type":"integer"},{"name":"value","type":"integer"}],
		 "return":{"type":"void"}}],
	"return":{"type":"list<String>"}}`

// TestDesignProblem covers the class-with-operations shape: LRU Cache is the canonical
// example and was declined outright before.
func TestDesignProblem(t *testing.T) {
	l := python3(t)
	lang, _ := Lookup("python3")

	dir := t.TempDir()
	solution := `
from collections import OrderedDict


class LRUCache:
    def __init__(self, capacity):
        self.cap = capacity
        self.d = OrderedDict()

    def get(self, key):
        if key not in self.d:
            return -1
        self.d.move_to_end(key)
        return self.d[key]

    def put(self, key, value):
        if key in self.d:
            self.d.move_to_end(key)
        self.d[key] = value
        if len(self.d) > self.cap:
            self.d.popitem(last=False)
`
	if err := os.WriteFile(filepath.Join(dir, "solution.py"), []byte(solution), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.Generate(context.Background(),
		Problem{Slug: "lru-cache", MetaData: lruMeta}, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// LeetCode's own example for problem 146.
	res, err := l.Run(context.Background(), dir, lang, []TestCase{{
		Input:    `["LRUCache","put","put","get","put","get","put","get","get","get"]` + "\n" + `[[2],[1,1],[2,2],[1],[3,3],[2],[4,4],[1],[3],[4]]`,
		Expected: "[null,null,null,1,null,-1,null,-1,3,4]",
	}}, RuleFor("lru-cache"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Passed() {
		t.Errorf("design problem failed: actual=%q err=%v", res.Cases[0].Actual, res.Cases[0].Err)
	}
}

func TestDesignMethodsIncludeConstructor(t *testing.T) {
	m, err := ParseMeta(lruMeta)
	if err != nil {
		t.Fatal(err)
	}
	methods, err := m.DesignMethods()
	if err != nil {
		t.Fatal(err)
	}
	if len(methods) != 3 {
		t.Fatalf("got %d methods, want constructor plus two", len(methods))
	}
	// The constructor is named after the class, which is how LeetCode lists it.
	if methods[0].Name != "LRUCache" {
		t.Errorf("first method = %q, want the constructor", methods[0].Name)
	}
	if len(methods[0].Params) != 1 {
		t.Errorf("constructor params = %v", methods[0].Params)
	}
}

// TestDesignDeclinedForLanguagesWithoutIt: Go's driver has no design shape, so it must
// decline rather than generate something that compiles into nonsense.
func TestDesignDeclinedForLanguagesWithoutIt(t *testing.T) {
	l := NewLocal()
	lang, _ := Lookup("golang")
	err := l.Generate(context.Background(),
		Problem{Slug: "lru-cache", MetaData: lruMeta}, lang, t.TempDir())
	if err == nil {
		t.Fatal("Go generated a design driver it does not have")
	}
}
