package solve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// Select prepares a slug in the configured workspace, or selects the actual local
// solution when arg names an existing file/folder (or is omitted for cwd).
// Existing local solutions and test cases are never rewritten by selection.
func Select(root, arg string, d *store.Detail, lang runner.Lang) (Layout, error) {
	path := arg
	if path == "" {
		path = "."
	}
	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) && arg != "" && filepath.Ext(arg) == "" && !strings.ContainsAny(arg, `/\`) {
		return Prepare(root, d, lang)
	}
	if err != nil {
		return Layout{}, fmt.Errorf("resolve solution: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Layout{}, err
	}
	dir, solution := abs, ""
	explicitFile := !fi.IsDir()
	if explicitFile {
		if !fi.Mode().IsRegular() {
			return Layout{}, fmt.Errorf("solution is not a regular file: %s", abs)
		}
		if !strings.EqualFold(filepath.Ext(abs), lang.Ext) {
			return Layout{}, fmt.Errorf("%s is not a %s solution file", abs, lang.Display)
		}
		dir, solution = filepath.Dir(abs), abs
	}
	for {
		if m := folderName.FindStringSubmatch(filepath.Base(dir)); m != nil && m[2] == d.Slug {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Layout{}, ErrNoProblem
		}
		dir = parent
	}
	if solution == "" {
		solution = filepath.Join(dir, lang.Filename())
	}
	meta, metaErr := runner.ParseMeta(d.MetaData)
	out := Layout{Dir: dir, Solution: solution, Meta: meta, Statement: Statement(d)}
	if fi, err := os.Stat(solution); errors.Is(err, os.ErrNotExist) {
		// Explicit files must still exist; only folder selection may scaffold.
		if explicitFile {
			return out, err
		}
		snippet, ok := d.Snippets[lang.Slug]
		if !ok {
			return out, fmt.Errorf("%s does not offer %s", d.Title, lang.Display)
		}
		s := runner.Scaffold{Lang: lang, Meta: meta, ID: d.NumericID, Title: d.Title, Difficulty: d.Difficulty}
		if err := seedSelectedFile(solution, s.File(snippet)); err != nil {
			return out, err
		}
	} else if err != nil {
		return out, err
	} else if !fi.Mode().IsRegular() {
		return out, fmt.Errorf("solution is not a regular file: %s", solution)
	}
	if metaErr == nil {
		n := len(meta.Params)
		if meta.IsDesign() {
			n = 2
		}
		cases := runner.ParseCases(d.ExampleTestcases, out.Statement, n)
		if len(cases) > 0 {
			if err := seedSelectedFile(filepath.Join(dir, workspace.TestcasesFile), runner.FormatCases(cases)); err != nil {
				return out, err
			}
		}
	}
	return out, nil
}

func seedSelectedFile(path, body string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.WriteString(body)
	return errors.Join(err, f.Close())
}
