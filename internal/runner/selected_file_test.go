package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExplicitCppMigratesOnlySelectedScaffold(t *testing.T) {
	engine, lang := cppLang(t)
	dir := filepath.Join(t.TempDir(), "0001-test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	engine.SolutionFile = filepath.Join(dir, "attempt.cpp")
	body := "#include \"leetui_driver.h\"\n// @leetui code=start\nclass Solution { public: int answer(int n) { return n + 1; } };\n// @leetui code=end\n"
	if err := os.WriteFile(engine.SolutionFile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "solution.cpp")
	if err := os.WriteFile(other, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Problem{Slug: "test", MetaData: `{"name":"answer","params":[{"name":"n","type":"integer"}],"return":{"type":"integer"}}`}
	ctx := context.Background()
	if err := engine.Generate(ctx, p, lang, dir); err != nil {
		t.Fatal(err)
	}
	res, err := engine.Run(ctx, dir, lang, []TestCase{{Input: "7", Expected: "8"}}, DefaultRule())
	if err != nil || !res.Passed() {
		t.Fatalf("legacy selected source = %+v, %v", res, err)
	}
	got, err := os.ReadFile(other)
	if err != nil || string(got) != body {
		t.Fatalf("unselected source changed: %q, %v", got, err)
	}
}

func TestLocalUsesExplicitSourceAcrossLanguages(t *testing.T) {
	for _, tc := range []struct{ slug, source, other string }{
		{"python3", "class Solution:\n    def answer(self, n): return n + 1\n", "class Solution:\n    def answer(self, n): return -1\n"},
		{"golang", "package main\nfunc answer(n int) int { return n + 1 }\n", "package main\nfunc answer(n int) int { return -1 }\n"},
		{"cpp", "class Solution { public: int answer(int n) { return n + 1; } };\n", "class Solution { public: int answer(int n) { return -1; } };\n"},
		{"javascript", "var answer = n => n + 1;\n", "var answer = n => -1;\n"},
		{"typescript", "function answer(n: number): number { return n + 1; }\n", "function answer(n: number): number { return -1; }\n"},
	} {
		for _, nested := range []bool{false, true} {
			t.Run(tc.slug+map[bool]string{true: "/nested", false: "/same-folder"}[nested], func(t *testing.T) {
				lang, _ := Lookup(tc.slug)
				engine := NewLocal()
				if !engine.Supports(lang) {
					t.Skip("toolchain unavailable")
				}
				dir := filepath.Join(t.TempDir(), "0001-test")
				sourceDir := dir
				if nested {
					sourceDir = filepath.Join(dir, "attempts")
				}
				if err := os.MkdirAll(sourceDir, 0o755); err != nil {
					t.Fatal(err)
				}
				other := filepath.Join(dir, lang.Filename())
				if err := os.WriteFile(other, []byte(tc.other), 0o644); err != nil {
					t.Fatal(err)
				}
				engine.SolutionFile = filepath.Join(sourceDir, "my answer"+lang.Ext)
				if err := os.WriteFile(engine.SolutionFile, []byte(tc.source), 0o644); err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				p := Problem{Slug: "test", MetaData: `{"name":"answer","params":[{"name":"n","type":"integer"}],"return":{"type":"integer"}}`}
				if err := engine.Generate(ctx, p, lang, dir); err != nil {
					t.Fatal(err)
				}
				result, err := engine.Run(ctx, dir, lang, []TestCase{{Input: "7", Expected: "8"}}, DefaultRule())
				if err != nil || !result.Passed() {
					t.Fatalf("selected source: %+v, %v", result, err)
				}
				for path, want := range map[string]string{other: tc.other, engine.SolutionFile: tc.source} {
					got, err := os.ReadFile(path)
					if err != nil || string(got) != want {
						t.Errorf("source changed at %s: %q, %v", path, got, err)
					}
				}
			})
		}
	}
}
