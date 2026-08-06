package tui

import (
	"os"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// The workbench strip.
//
// leetui is the LeetCode side of the desk: browse, read, run, submit. The CODE happens in
// whatever the user already has open — nvim in the next tmux pane, a VS Code terminal, an
// editor on another monitor. leetui never has to be the editor, and mostly should not try.
//
// That arrangement has one hard requirement, and it was missing: leetui must SAY WHERE THE
// FILE IS. Without the path there is nothing to open, and the whole model collapses back
// into pressing `e` and hoping.
//
// It also says whether saves re-run. A watcher that works silently is indistinguishable
// from one that is off, and the difference decides whether you switch back here after
// every save or never think about leetui again.

// workbenchHeight is the strip's full height: bezel, path, state, bezel.
const workbenchHeight = 4

// viewWorkbench names the file for the selected problem and language.
func (m Model) viewWorkbench(w int) string {
	f := components.Frame{
		Title:  "solution",
		Right:  m.lang.Display,
		Width:  w,
		Height: workbenchHeight,
	}

	path := m.solutionPath()
	if path == "" {
		return f.Render(" " + theme.Meta.Render("Select a problem.") + "\n")
	}

	var b strings.Builder
	b.WriteString(" " + theme.Body.Render(fitPath(path, f.InnerWidth()-2)) + "\n")
	b.WriteString(" " + m.workbenchState(path) + "\n")
	return f.Render(b.String())
}

// workbenchState is the one line that says what will happen next.
func (m Model) workbenchState(path string) string {
	exists := false
	if _, err := os.Stat(path); err == nil {
		exists = true
	}

	switch {
	case m.running:
		return theme.Label.Render("running…")

	case !exists:
		// The file appears on the first edit, run, or submit. Saying so is better than a
		// path that is not there yet with no explanation.
		return theme.Meta.Render("not created yet · e or r makes it")

	case !m.cfg.WatchSolution:
		// Not "watching off": skimmed, that reads as watching. The states here are
		// glanced at, never studied.
		return theme.Meta.Render("watch off · r to run")

	case !m.engine.Supports(m.lang):
		// No local driver, so a save has nothing to re-run. Saying "watching" here would
		// promise something that never happens.
		return theme.Meta.Render("judge only · s to submit")

	default:
		return theme.Label.Render("watching") +
			theme.Meta.Render(" · saves re-run the tests")
	}
}

// solutionPath is the file for the selected problem in the current language, whether or
// not it exists yet.
//
// Asks the workspace rather than assembling the name here. A second copy of the folder
// convention would eventually drift from workspace.FolderName and show a path that is not
// where the file actually is — which is the one thing this strip must never do.
func (m Model) solutionPath() string {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return ""
	}
	ws, err := workspace.New(m.cfg.Workspace)
	if err != nil {
		return ""
	}
	row := m.rows[m.cursor]
	return ws.Path(row.NumericID, row.Slug, m.lang.Filename())
}

// fitPath shortens a path from the LEFT, keeping the filename.
//
// The end of the path is what identifies the file; the middle of a long workspace root is
// what can be spared. Truncating from the right would leave "…/0001-two-s".
func fitPath(path string, w int) string {
	if w <= 0 {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+"/") {
		path = "~" + strings.TrimPrefix(path, home)
	}
	r := []rune(path)
	if len(r) <= w {
		return path
	}
	if w <= 1 {
		return "…"
	}
	return "…" + string(r[len(r)-(w-1):])
}
