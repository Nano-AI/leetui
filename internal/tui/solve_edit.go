package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// Where the editor opens.
//
// leetui and a terminal editor both want the whole terminal, and handing it over makes
// the statement disappear at the moment you start writing code against it. Three routes
// out, best first:
//
//	PANE      inside tmux/zellij/WezTerm/Kitty: open the editor beside leetui. The
//	          statement stays on screen and the watcher re-runs on every save.
//	DETACHED  a GUI editor already opens its own window, so blocking leetui behind it
//	          buys nothing and costs the statement and the watcher.
//	TAKEOVER  a terminal editor with nowhere else to go. README.md opens alongside so
//	          the problem is at least readable, and the tests run when it exits.
//
// Only the last one suspends leetui, and it is the only one that has to.
type editRoute int

const (
	routePane editRoute = iota
	routeDetached
	routeTakeover
)

// editPlan is a resolved decision about how to open one file.
type editPlan struct {
	route    editRoute
	editor   editor.Editor
	splitter editor.Splitter

	// argv is the editor invocation: command first, then arguments.
	argv []string
	// dir is the problem folder, which the new pane starts in.
	dir string
}

// planEdit decides how to open file, given what this terminal can do.
func planEdit(cfg config.Config, ed editor.Editor, dir, file string) editPlan {
	p := editPlan{editor: ed, dir: dir}

	if ed.GUI {
		// A GUI editor never needs the terminal, so it never takes it.
		p.route = routeDetached
		cmd, args := ed.LaunchDetached(file)
		p.argv = append([]string{cmd}, args...)
		return p
	}

	if cfg.EditorPane {
		if s, ok := editor.DetectSplitter(); ok {
			p.route, p.splitter = routePane, s
			cmd, args := ed.LaunchDetached(file)
			p.argv = append([]string{cmd}, args...)
			return p
		}
	}

	// Nowhere to put a second pane: take the terminal, and bring the statement along.
	p.route = routeTakeover
	cmd, args := ed.Launch(file, statementArgs(cfg, ed, dir)...)
	p.argv = append([]string{cmd}, args...)
	if split := ed.SplitArgs(); len(split) > 0 && len(p.argv) > 2 {
		// The split flag has to precede the filenames.
		p.argv = append(append([]string{p.argv[0]}, split...), p.argv[1:]...)
	}
	return p
}

// statementArgs returns README.md when it exists and the editor can show it beside the
// solution. An editor with no known split flag gets nothing, because a second file it
// opens stacked is a buffer in the way rather than a statement to read.
func statementArgs(cfg config.Config, ed editor.Editor, dir string) []string {
	if !cfg.OpenStatement || len(ed.SplitArgs()) == 0 {
		return nil
	}
	readme := filepath.Join(dir, workspace.ReadmeFile)
	if _, err := os.Stat(readme); err != nil {
		return nil
	}
	return []string{readme}
}

// note is what the status line says once the editor is open.
//
// Each route implies a different next move, and the one the user needs to hear is the one
// that is not obvious: a pane means they never have to come back here.
func (p editPlan) note() string {
	switch p.route {
	case routePane:
		return "Opened " + p.editor.Name + " in a " + p.splitter.Name +
			" pane. Save to re-run the tests — no need to switch back."
	case routeDetached:
		return "Opened " + p.editor.Name + ". Save to re-run the tests."
	default:
		return ""
	}
}

// fallbackNote explains a pane that could not be opened, without pretending it worked.
func fallbackNote(name string, err error) string {
	msg := strings.TrimSpace(err.Error())
	return "Could not open a " + name + " pane (" + msg + "). Set editor_pane = false to stop trying."
}
