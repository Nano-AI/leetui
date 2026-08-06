package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Problem is what the workspace needs to lay out a folder.
type Problem struct {
	ID    int
	Slug  string
	Title string

	// Statement is the problem body as markdown, for README.md.
	Statement string

	// Difficulty and URL go in the README header.
	Difficulty string
	URL        string
}

// Create lays out a problem's folder, writing anything that does not exist yet.
//
// It is safe to call repeatedly: the README is refreshed from the statement, but the
// solution file, notes, and testcases are only ever created, never overwritten. Losing
// a user's work to a re-sync would be unforgivable, so the asymmetry is deliberate.
func (w Workspace) Create(p Problem) (string, error) {
	dir := w.Dir(p.ID, p.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create problem folder: %w", err)
	}

	if err := w.writeReadme(dir, p); err != nil {
		return dir, err
	}
	if err := createIfMissing(filepath.Join(dir, NotesFile), notesTemplate(p)); err != nil {
		return dir, err
	}
	return dir, nil
}

// writeReadme regenerates README.md. This one IS overwritten — it is derived from the
// statement and holds nothing the user typed.
func (w Workspace) writeReadme(dir string, p Problem) error {
	header := fmt.Sprintf("# %d. %s\n\n", p.ID, p.Title)
	if p.Difficulty != "" {
		header += fmt.Sprintf("**%s**", p.Difficulty)
		if p.URL != "" {
			header += fmt.Sprintf("  ·  [leetcode.com](%s)", p.URL)
		}
		header += "\n\n---\n\n"
	}

	body := p.Statement
	if body == "" {
		body = "_Statement not synced yet._\n"
	}

	if err := os.WriteFile(filepath.Join(dir, ReadmeFile), []byte(header+body), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	return nil
}

func notesTemplate(p Problem) string {
	return fmt.Sprintf("# %d. %s — notes\n\nApproach:\n\nComplexity:\n\n- time:\n- space:\n",
		p.ID, p.Title)
}

// WriteSolution creates the solution file from a starter snippet.
//
// An existing file is left completely alone and reported via created=false. This is the
// single most important guarantee in this package: `leetui` must never be the reason
// someone loses a solution they were part-way through.
func (w Workspace) WriteSolution(id int, slug, filename, snippet string) (path string, created bool, err error) {
	dir := w.Dir(id, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, fmt.Errorf("create problem folder: %w", err)
	}
	path = filepath.Join(dir, filename)

	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return path, false, fmt.Errorf("check %s: %w", filename, err)
	}

	if err := os.WriteFile(path, []byte(snippet), 0o644); err != nil {
		return path, false, fmt.Errorf("write %s: %w", filename, err)
	}
	return path, true, nil
}

// WriteTestcases seeds testcases.txt, leaving an existing file alone — the user may
// have added cases of their own.
func (w Workspace) WriteTestcases(id int, slug, content string) (string, error) {
	path := w.Path(id, slug, TestcasesFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create problem folder: %w", err)
	}
	return path, createIfMissing(path, content)
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

// ReadFile reads a file from a problem's folder.
func (w Workspace) ReadFile(id int, slug, name string) (string, error) {
	b, err := os.ReadFile(w.Path(id, slug, name))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ModTime reports when a file last changed, for detecting an external editor's save.
// A missing file returns the zero time and no error.
func (w Workspace) ModTime(id int, slug, name string) (time.Time, error) {
	fi, err := os.Stat(w.Path(id, slug, name))
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

// Solutions lists the solution files present for a problem, by filename.
func (w Workspace) Solutions(id int, slug string) []string {
	entries, err := os.ReadDir(w.Dir(id, slug))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && len(n) > 9 && n[:9] == "solution." {
			out = append(out, n)
		}
	}
	return out
}

func createIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
