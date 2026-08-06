package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const twoSumMeta = `{"name":"twoSum","params":[{"name":"nums","type":"integer[]"},
	{"name":"target","type":"integer"}],"return":{"type":"integer[]","size":2}}`

func python3(t *testing.T) *Local {
	t.Helper()
	l := NewLocal()
	lang, _ := Lookup("python3")
	if !l.Supports(lang) {
		t.Skip("python3 not on PATH")
	}
	return l
}

// scaffolded wraps a bare solution body in the file leetui actually writes, so the
// compiled tests prove the scaffolding itself builds. A file that only compiles without
// its imports and package clause is not the file the user edits.
func scaffolded(t *testing.T, lang Lang, meta, solution string) string {
	t.Helper()
	m, err := ParseMeta(meta)
	if err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	return Scaffold{Lang: lang, Meta: m, ID: 1, Title: "Test", URL: "https://leetcode.com/"}.
		File(solution)
}

// generate lays out a problem folder with a solution and returns the directory.
func generate(t *testing.T, l *Local, slug, meta, solution string) string {
	t.Helper()
	dir := t.TempDir()
	lang, _ := Lookup("python3")
	if err := os.WriteFile(filepath.Join(dir, "solution.py"),
		[]byte(scaffolded(t, lang, meta, solution)), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Problem{Slug: slug, MetaData: meta}
	if err := l.Generate(context.Background(), p, lang, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return dir
}

func run(t *testing.T, l *Local, dir, slug string, cases []TestCase) Result {
	t.Helper()
	lang, _ := Lookup("python3")
	res, err := l.Run(context.Background(), dir, lang, cases, RuleFor(slug))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

func TestSupportsAndToolchain(t *testing.T) {
	l := NewLocal()

	java, _ := Lookup("java")
	if l.Supports(java) {
		t.Error("java has no vendored driver and must not report as locally runnable")
	}
	if l.MissingToolchain(java) != "" {
		t.Error("a language with no driver should not blame a missing toolchain")
	}

	// Rust has no driver yet, so it must report as judge-only rather than blaming a
	// missing toolchain — "install rustup" would be the wrong advice.
	rust, _ := Lookup("rust")
	if rust.Local {
		t.Error("rust is marked local but has no driver in drivers/")
	}
	if l.MissingToolchain(rust) != "" {
		t.Errorf("MissingToolchain(rust) = %q; a driverless language must not blame a toolchain",
			l.MissingToolchain(rust))
	}

	// Python has a driver, so on a machine with python3 it must actually run.
	py, _ := Lookup("python3")
	if !py.Local {
		t.Error("python3 has a vendored driver and should be marked local")
	}
}
