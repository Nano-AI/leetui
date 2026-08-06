package solve

import (
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// The two earned exceptions to never-overwrite. Each proves its case before writing;
// see the package comment and D-010a / D-014.

// writeSolution creates the solution file, and adds scaffolding to one that predates it.
//
// A folder created by an earlier leetui holds a bare LeetCode snippet: no imports, no
// package clause, no driver types. Opening it gives a buffer of unresolved symbols, which
// is the whole problem scaffolding exists to fix — so leaving those files alone would fix
// the feature only for problems the user has not started yet.
//
// The upgrade is purely ADDITIVE: Wrap puts the existing content inside the markers
// verbatim and adds lines above and below it. Nothing the user typed can be lost, which
// is what makes rewriting a file they may have edited defensible. A file that already has
// markers is left completely alone.
func writeSolution(ws workspace.Workspace, d *store.Detail, lang runner.Lang, s runner.Scaffold) (string, error) {
	snippet := d.Snippets[lang.Slug]
	path, created, err := ws.WriteSolution(d.NumericID, d.Slug, lang.Filename(), s.File(snippet))
	if err != nil || created {
		return path, err
	}

	existing, err := ws.ReadFile(d.NumericID, d.Slug, lang.Filename())
	if err != nil {
		return path, nil
	}
	wrapped, changed := s.Wrap(existing)
	if !changed {
		return path, nil
	}
	return path, ws.ReplaceSolution(d.NumericID, d.Slug, lang.Filename(), wrapped)
}

// seedCases writes testcases.txt, and repairs one written by an older leetui.
//
// Existing cases are never replaced — the user may have added their own. The single
// exception is a file where EVERY case has an empty expected answer, which is what the
// broken scrape produced: it holds nothing a person would have typed, and leaving it
// would make every future run report that there was nothing to check against.
func seedCases(ws workspace.Workspace, d *store.Detail, cases []runner.TestCase) {
	if len(cases) == 0 {
		return
	}
	formatted := runner.FormatCases(cases)

	existing, err := ws.ReadFile(d.NumericID, d.Slug, workspace.TestcasesFile)
	if err != nil {
		_, _ = ws.WriteTestcases(d.NumericID, d.Slug, formatted)
		return
	}
	if runner.HasExpected(runner.LoadCases(existing)) || !runner.HasExpected(cases) {
		return
	}
	_, _ = ws.ReplaceTestcases(d.NumericID, d.Slug, formatted)
}
