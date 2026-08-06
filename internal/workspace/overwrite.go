package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// The writes that are allowed to replace something.
//
// Everything in files.go creates or leaves alone. These three do not, so they live apart
// where the exception is visible: two earned overwrites (D-010a, D-014) and the workspace
// .gitignore. A new writer belongs in files.go unless it can justify being here.

// generatedIgnores are the files leetui writes into a problem folder that nobody should
// commit: drivers, compiled binaries, and interpreter caches. Kept here rather than in
// each language's generator so the list is reviewable in one place.
const generatedIgnores = `# Written by leetui. Solutions, notes, and testcases are yours; these are not.
__pycache__/
_leetui_*
leetui_driver.*
leetui_main.*
leetui_bin
go.mod
go.sum
`

// EnsureGitignore writes a .gitignore at the workspace root, if there is not one already.
//
// The generated drivers and their build output live beside the solution (D-010), so a
// workspace that is a git repository would otherwise stage a compiled binary and a
// __pycache__ on the first commit. Writing this at layout time means the repository is
// clean before Phase 4 ever runs `git add`.
//
// Never overwrites: a user who has edited it has made a decision.
func (w Workspace) EnsureGitignore() error {
	return createIfMissing(filepath.Join(w.Root, ".gitignore"), generatedIgnores)
}

// ReplaceSolution overwrites a solution file.
//
// The second earned exception to never-overwrite, and the more dangerous one, so the
// caller must prove the file holds no work before calling — the only accepted proof is
// that it is byte-identical to LeetCode's starter snippet. See writeSolution in the TUI.
// Every other write goes through WriteSolution, which never overwrites.
func (w Workspace) ReplaceSolution(id int, slug, filename, contents string) error {
	path := w.Path(id, slug, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create problem folder: %w", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}
	return nil
}

// ReplaceTestcases overwrites testcases.txt.
//
// This is the ONE exception to the never-overwrite rule, and callers must earn it. It
// exists for a single case: a file leetui itself wrote with no expected answers, which
// carries nothing a user could have typed. The caller checks that before calling — see
// the repair in the TUI's prepare. Anything else must go through WriteTestcases.
func (w Workspace) ReplaceTestcases(id int, slug, content string) (string, error) {
	path := w.Path(id, slug, TestcasesFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create problem folder: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return path, fmt.Errorf("write %s: %w", TestcasesFile, err)
	}
	return path, nil
}
