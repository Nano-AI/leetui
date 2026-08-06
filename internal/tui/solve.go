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
	"github.com/Nano-AI/leetui/internal/tui/theme"
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

	file, _, err = ws.WriteSolution(d.NumericID, d.Slug, lang.Filename(), snippet)
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

// runLocalCmd generates a driver and runs the solution against its test cases.
func (m Model) runLocalCmd(d *store.Detail, lang runner.Lang) tea.Cmd {
	engine := m.engine
	model := m

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		dir, _, err := model.prepare(ctx, d, lang)
		if err != nil {
			return runFinishedMsg{err: err}
		}

		p := runner.Problem{Slug: d.Slug, Title: d.Title, MetaData: d.MetaData}
		if err := engine.Generate(ctx, p, lang, dir); err != nil {
			return runFinishedMsg{err: err}
		}

		ws, err := workspace.New(model.cfg.Workspace)
		if err != nil {
			return runFinishedMsg{err: err}
		}
		raw, err := ws.ReadFile(d.NumericID, d.Slug, workspace.TestcasesFile)
		if err != nil {
			return runFinishedMsg{err: fmt.Errorf("read test cases: %w", err)}
		}
		cases := runner.LoadCases(raw)
		if len(cases) == 0 {
			return runFinishedMsg{err: fmt.Errorf("no test cases in %s", workspace.TestcasesFile)}
		}

		res, err := engine.Run(ctx, dir, lang, cases, runner.RuleFor(d.Slug))
		return runFinishedMsg{slug: d.Slug, result: res, err: err}
	}
}

// submitCmd sends the solution to the judge and polls until it resolves.
//
// Progress arrives as separate messages so the queue's flap can animate through
// PENDING -> JUDGING -> the verdict rather than jumping straight to the answer.
func (m Model) submitCmd(d *store.Detail, lang runner.Lang, flapID int) tea.Cmd {
	client := m.client
	model := m

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		_, file, err := model.prepare(ctx, d, lang)
		if err != nil {
			return judgeMsg{flapID: flapID, err: err}
		}
		code, err := os.ReadFile(file)
		if err != nil {
			return judgeMsg{flapID: flapID, err: fmt.Errorf("read solution: %w", err)}
		}

		// questionId, never the frontend id — they are different numbers and the judge
		// takes the internal one.
		if d.QuestionID == "" {
			return judgeMsg{flapID: flapID, err: fmt.Errorf("%s has no questionId cached; reopen it first", d.Slug)}
		}

		id, err := client.Submit(ctx, leetcode.Submission{
			Slug: d.Slug, QuestionID: d.QuestionID, Lang: lang.Slug, Code: string(code),
		})
		if err != nil {
			return judgeMsg{flapID: flapID, err: err}
		}

		j, err := client.Poll(ctx, id, nil)
		return judgeMsg{flapID: flapID, judgement: j, err: err}
	}
}

// verdictOf maps the judge's numeric result onto the theme's verdict vocabulary.
func verdictOf(j leetcode.Judgement) theme.Verdict {
	switch j.StatusCode {
	case leetcode.VerdictAccepted:
		return theme.Accepted
	case leetcode.VerdictWrongAnswer:
		return theme.WrongAnswer
	case leetcode.VerdictTimeLimitExceeded:
		return theme.TimeLimitExceeded
	case leetcode.VerdictMemoryLimitExceeded:
		return theme.MemoryLimitExceeded
	case leetcode.VerdictCompileError:
		return theme.CompileError
	case leetcode.VerdictRuntimeError:
		return theme.RuntimeError
	default:
		return theme.Judging
	}
}
