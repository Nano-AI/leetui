package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// The solve loop: lay out files, edit, run, submit.
//
// Every step needs the problem's full detail (statement, snippets, metaData), which may
// not be cached yet. Rather than each action fetching separately, prepare() gets it once
// and the actions build on that.

// prepare makes sure a problem's folder exists with a solution file for the chosen
// language, returning the directory and the file.
func (m Model) prepare(ctx context.Context, d *store.Detail, lang runner.Lang) (dir, file string, err error) {
	ws, err := workspace.New(m.cfg.Workspace)
	if err != nil {
		return "", "", err
	}

	statement := m.detailMD
	if statement == "" {
		statement = "_Open this problem in leetui to sync its statement._\n"
	}

	dir, err = ws.Create(workspace.Problem{
		ID: d.NumericID, Slug: d.Slug, Title: d.Title,
		Statement: statement, Difficulty: d.Difficulty,
		URL: leetcode.BaseURL + "/problems/" + d.Slug + "/",
	})
	if err != nil {
		return "", "", err
	}

	snippet, ok := d.Snippets[lang.Slug]
	if !ok {
		return dir, "", fmt.Errorf("%s does not offer %s", d.Title, lang.Display)
	}

	file, _, err = ws.WriteSolution(d.NumericID, d.Slug, lang.Filename(), lang.SolutionFile(snippet))
	if err != nil {
		return dir, "", err
	}

	// Seed test cases from the examples. Existing cases are never replaced — the user
	// may have added their own.
	meta, mErr := runner.ParseMeta(d.MetaData)
	if mErr == nil {
		cases := runner.ParseCases(d.ExampleTestcases, statement, len(meta.Params))
		if len(cases) > 0 {
			_, _ = ws.WriteTestcases(d.NumericID, d.Slug, runner.FormatCases(cases))
		}
	}
	return dir, file, nil
}

// editCmd opens the solution in the user's editor.
//
// tea.ExecProcess hands the terminal over and takes it back cleanly — leaving the alt
// screen and mouse mode on while a full-screen editor runs would corrupt both.
func (m Model) editCmd(d *store.Detail, lang runner.Lang) tea.Cmd {
	cfg := m.cfg
	model := m

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, file, err := model.prepare(ctx, d, lang)
		if err != nil {
			return statusMsg{text: err.Error(), isError: true}
		}
		ed := resolveEditor(cfg.Editor)
		cmd, args := ed.Launch(file)
		return editReadyMsg{name: ed.Name, argv: append([]string{cmd}, args...)}
	}
}

// resolveEditor decides which editor to launch, and with which arguments.
//
// Config wins over the environment so leetui can be pointed somewhere specific without
// changing the shell. The fallback chain matters: $EDITOR and $VISUAL are both commonly
// unset (D-012).
//
// Resolving through the editor package rather than returning a bare command name is
// what carries a GUI editor's blocking flag. Launch "code" without --wait and it forks,
// returns instantly, and the TUI repaints over a file nobody has typed in yet.
func resolveEditor(configured string) editor.Editor {
	if configured != "" {
		if e, ok := editor.Lookup(configured); ok {
			return e
		}
		// Configured but unknown: honour it verbatim rather than overriding the user.
		return editor.FromCommand(configured)
	}

	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			if e, ok := editor.Lookup(v); ok {
				return e
			}
			return editor.FromCommand(v)
		}
	}

	// Nothing configured anywhere: prefer a terminal editor, which Detect sorts first.
	if found := editor.Detect(); len(found) > 0 {
		return found[0]
	}
	return editor.FromCommand("vi")
}
