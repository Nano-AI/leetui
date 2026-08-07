// Package workspace owns the on-disk layout of solved problems.
//
// Layout is problem-first (D-010):
//
//	<workspace>/
//	  0001-two-sum/
//	    README.md        problem statement, regenerated on sync
//	    solution.py      one per language attempted
//	    solution.go
//	    notes.md         the user's own notes — NEVER overwritten
//	    testcases.txt    editable, seeded from the statement's examples
//
// GitHub renders README.md per directory, so browsing the repo shows each problem
// inline. Zero-padding sorts correctly. Everything about one problem stays together.
//
// This layout is effectively permanent: changing it once solutions are committed is a
// history-rewriting migration. Treat it as fixed.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Filenames inside a problem folder.
const (
	ReadmeFile    = "README.md"
	NotesFile     = "notes.md"
	TestcasesFile = "testcases.txt"
)

// GlobalsDir holds the parts of a driver that are identical for every problem.
//
// They used to be written into each problem folder, which put ~19KB of byte-identical
// scaffolding beside every solution and buried the four files you actually edit under
// four you never open.
//
// A sibling directory rather than a shared path elsewhere, because the include stays
// RELATIVE — `#include "../globals/leetui_driver.h"` resolves with no -I flag and no
// compile_commands.json, so clangd follows it with zero configuration. That was the
// objection to sharing them at all, and a relative path answers it.
const GlobalsDir = "globals"

// Globals returns the shared-driver directory, creating it if needed.
func (w Workspace) Globals() (string, error) {
	dir := filepath.Join(w.Root, GlobalsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create globals dir: %w", err)
	}
	return dir, nil
}

// GlobalRef is how a generated file refers to a shared driver from inside a problem
// folder. One level up, then down.
func GlobalRef(name string) string { return "../" + GlobalsDir + "/" + name }

// Workspace is a root directory holding problem folders.
type Workspace struct {
	Root string
}

// New returns a Workspace rooted at dir, expanding a leading ~.
func New(dir string) (Workspace, error) {
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return Workspace{}, fmt.Errorf("expand ~: %w", err)
		}
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace path: %w", err)
	}
	return Workspace{Root: abs}, nil
}

// slugSafe strips anything that has no business in a directory name.
//
// LeetCode slugs are already lowercase-hyphenated, but this is the boundary between
// remote data and the filesystem, so it is validated rather than trusted.
var slugSafe = regexp.MustCompile(`[^a-z0-9-]+`)

// FolderName is the directory for a problem, e.g. "0146-lru-cache".
//
// Problems with non-numeric IDs (the LCP/LCS contest series) get id 0 and sort to the
// front, which is correct: they have no place in the numbered sequence.
func FolderName(id int, slug string) string {
	clean := slugSafe.ReplaceAllString(strings.ToLower(slug), "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "problem"
	}
	return fmt.Sprintf("%04d-%s", id, clean)
}

// Dir returns the absolute path to a problem's folder. It does not create it.
func (w Workspace) Dir(id int, slug string) string {
	return filepath.Join(w.Root, FolderName(id, slug))
}

// Path returns a file inside a problem's folder.
func (w Workspace) Path(id int, slug, file string) string {
	return filepath.Join(w.Dir(id, slug), file)
}

// Exists reports whether a problem's folder has been created.
func (w Workspace) Exists(id int, slug string) bool {
	fi, err := os.Stat(w.Dir(id, slug))
	return err == nil && fi.IsDir()
}
