package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/Nano-AI/leetui/internal/workspace"
	tea "github.com/charmbracelet/bubbletea"
)

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
