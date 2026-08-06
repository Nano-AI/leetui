// Package solve lays out a problem's folder on disk.
//
// This is the seam between a frontend and the filesystem. The TUI and the CLI both need
// exactly the same thing before editing, running, or submitting — a folder, a README, a
// scaffolded solution file, and seeded test cases — and neither of them should own it.
//
// Everything here is idempotent. Prepare runs on every edit, run, and submit, so a second
// call must be a no-op rather than a second write.
//
// THE RULE: leetui must never be the reason someone loses work. Two narrow exceptions
// exist (D-010a, D-014) and each proves its case before writing.
package solve

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// Layout is where a prepared problem lives.
type Layout struct {
	// Dir is the problem folder.
	Dir string
	// Solution is the solution file for the chosen language.
	Solution string
	// Statement is the problem body as markdown, already converted.
	Statement string
	// Meta is the parsed metaData, zero when the problem has none.
	Meta runner.Meta
}

// Prepare lays out a problem's folder and returns where everything went.
func Prepare(root string, d *store.Detail, lang runner.Lang) (Layout, error) {
	ws, err := workspace.New(root)
	if err != nil {
		return Layout{}, err
	}

	// Keep the repository clean before Phase 4 ever runs `git add`. Best effort: a
	// workspace that cannot take a .gitignore is not a reason to refuse to lay out a
	// problem.
	_ = ws.EnsureGitignore()

	statement := Statement(d)
	dir, err := ws.Create(workspace.Problem{
		ID: d.NumericID, Slug: d.Slug, Title: d.Title,
		Statement: statement, Difficulty: d.Difficulty,
		URL: leetcode.BaseURL + "/problems/" + d.Slug + "/",
	})
	if err != nil {
		return Layout{}, err
	}
	out := Layout{Dir: dir, Statement: statement}

	if _, ok := d.Snippets[lang.Slug]; !ok {
		return out, fmt.Errorf("%s does not offer %s", d.Title, lang.Display)
	}

	// metaData drives both the scaffolding and the test cases. A problem without it can
	// still be edited and submitted — it just gets a plainer file and no seeded cases.
	meta, metaErr := runner.ParseMeta(d.MetaData)
	out.Meta = meta

	scaffold := runner.Scaffold{
		Lang: lang, Meta: meta,
		ID: d.NumericID, Title: d.Title, Difficulty: d.Difficulty,
		URL: leetcode.BaseURL + "/problems/" + d.Slug + "/",
	}
	out.Solution, err = writeSolution(ws, d, lang, scaffold)
	if err != nil {
		return out, err
	}

	if metaErr == nil {
		seedCases(ws, d, runner.ParseCases(d.ExampleTestcases, statement, len(meta.Params)))
	}
	return out, nil
}

// Statement converts a problem's HTML body to markdown.
//
// Converted HERE rather than reused from a rendered pane. Glamour's output is wrapped to
// a pane width and full of ANSI escape codes; the README is a file people open in an
// editor, and ParseCases scrapes expected answers with a line-anchored regex that cannot
// match a line beginning with a colour code.
func Statement(d *store.Detail) string {
	if strings.TrimSpace(d.Content) == "" {
		return "_Open this problem in leetui to sync its statement._\n"
	}
	doc, err := render.HTMLToMarkdown(d.Content)
	if err != nil {
		return "_Could not render this statement._\n"
	}
	return doc.Markdown
}
