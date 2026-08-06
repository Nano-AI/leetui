package solve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// Run generates the driver and executes the solution against its test cases.
//
// Shared by the TUI and the CLI. The order matters and is easy to get wrong: the driver
// has to exist before the build, and the cases are read from DISK rather than from the
// problem's example data — a user who added a case to testcases.txt expects it to run.
func Run(ctx context.Context, engine runner.Engine, dir string, d *store.Detail, lang runner.Lang) (runner.Result, error) {
	p := runner.Problem{Slug: d.Slug, Title: d.Title, MetaData: d.MetaData}
	if err := engine.Generate(ctx, p, lang, dir); err != nil {
		return runner.Result{}, err
	}

	cases, err := Cases(dir)
	if err != nil {
		return runner.Result{}, err
	}
	if len(cases) == 0 {
		return runner.Result{}, fmt.Errorf("no test cases in %s", workspace.TestcasesFile)
	}
	return engine.Run(ctx, dir, lang, cases, runner.RuleFor(d.Slug))
}

// Cases reads a problem folder's test cases.
//
// Read by directory rather than by id and slug, so a caller that already has the folder
// does not have to reconstruct its name.
func Cases(dir string) ([]runner.TestCase, error) {
	raw, err := os.ReadFile(filepath.Join(dir, workspace.TestcasesFile))
	if err != nil {
		return nil, fmt.Errorf("read test cases: %w", err)
	}
	return runner.LoadCases(string(raw)), nil
}

// Dir returns a problem's folder without creating anything.
func Dir(root string, d *store.Detail) (string, error) {
	ws, err := workspace.New(root)
	if err != nil {
		return "", err
	}
	return ws.Dir(d.NumericID, d.Slug), nil
}
