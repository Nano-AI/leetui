// Package editor finds the editors installed on this machine.
//
// leetui delegates editing rather than embedding one (D-012): writing a text editor is
// a project, not a feature, and users already have their config and LSP set up.
//
// The wrinkle is GUI editors. A terminal editor takes over the terminal and exits when
// you close the file, which is exactly the contract tea.ExecProcess wants. A GUI editor
// forks and returns instantly, so leetui would resume before a single character was
// typed. Every GUI entry here carries the flag that makes it block.
package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// Editor is an installed editor.
type Editor struct {
	// Name is what the picker shows.
	Name string
	// Command is the executable, resolved to an absolute path when found outside PATH.
	Command string
	// Args precede the filename, e.g. --wait.
	Args []string
	// GUI marks editors that open a window. They need a blocking flag to be usable.
	GUI bool
}

// Launch returns the command and full argument list for editing path.
//
// Extra paths are opened alongside the first. Terminal editors split their window for
// them, which is how the problem statement ends up next to the solution.
func (e Editor) Launch(path string, also ...string) (string, []string) {
	args := append(append([]string{}, e.Args...), path)
	return e.Command, append(args, also...)
}

// LaunchDetached is Launch WITHOUT the blocking flag.
//
// The --wait flags exist so tea.ExecProcess does not resume the TUI over a file nobody
// has typed in. When leetui is staying on screen instead — in another pane, or beside a
// GUI window — blocking is exactly wrong: it would freeze the statement and the file
// watcher for as long as the editor is open.
func (e Editor) LaunchDetached(path string, also ...string) (string, []string) {
	args := make([]string, 0, len(e.Args)+1+len(also))
	for _, a := range e.Args {
		if a == "--wait" || a == "-w" {
			continue
		}
		args = append(args, a)
	}
	return e.Command, append(append(args, path), also...)
}

// SplitArgs returns the flags that make a terminal editor open its files side by side.
//
// Only editors whose split flag is known get one. Passing a second file to an editor that
// does not understand the flag would open it stacked or, worse, not at all.
func (e Editor) SplitArgs() []string {
	switch base := filepath.Base(e.Command); base {
	case "nvim", "vim", "vi", "mvim", "gvim":
		return []string{"-O"} // vertical splits, one per file
	case "hx":
		return []string{"--vsplit"}
	default:
		return nil
	}
}

// candidate is a known editor and where to look for it.
type candidate struct {
	name string
	bin  string
	args []string
	gui  bool
	// macApp is a bundled CLI helper that is often not on PATH.
	macApp string
}

// known lists the editors leetui can detect, terminal ones first.
//
// The --wait flags are not optional. Without them a GUI editor returns immediately and
// the TUI redraws over a file the user has not touched yet.
var known = []candidate{
	// Terminal editors: they block by nature.
	{name: "Neovim", bin: "nvim"},
	{name: "Vim", bin: "vim"},
	{name: "Helix", bin: "hx"},
	{name: "Kakoune", bin: "kak"},
	{name: "Emacs", bin: "emacs"},
	{name: "Emacs (terminal)", bin: "emacsclient", args: []string{"-nw"}},
	{name: "Micro", bin: "micro"},
	{name: "Nano", bin: "nano"},
	{name: "vi", bin: "vi"},

	// GUI editors: each needs its blocking flag.
	{name: "VS Code", bin: "code", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"},
	{name: "VS Code Insiders", bin: "code-insiders", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code-insiders"},
	{name: "Cursor", bin: "cursor", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Cursor.app/Contents/Resources/app/bin/cursor"},
	{name: "Windsurf", bin: "windsurf", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Windsurf.app/Contents/Resources/app/bin/windsurf"},
	{name: "Zed", bin: "zed", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Zed.app/Contents/MacOS/cli"},
	{name: "Sublime Text", bin: "subl", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"},
	{name: "BBEdit", bin: "bbedit", args: []string{"--wait"}, gui: true},
	{name: "TextMate", bin: "mate", args: []string{"--wait"}, gui: true},

	// JetBrains. Their launchers all take --wait; which ones exist depends on the
	// Toolbox install, so each is probed separately.
	{name: "IntelliJ IDEA", bin: "idea", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/IntelliJ IDEA.app/Contents/MacOS/idea"},
	{name: "PyCharm", bin: "pycharm", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/PyCharm.app/Contents/MacOS/pycharm"},
	{name: "GoLand", bin: "goland", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/GoLand.app/Contents/MacOS/goland"},
	{name: "CLion", bin: "clion", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/CLion.app/Contents/MacOS/clion"},
	{name: "WebStorm", bin: "webstorm", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/WebStorm.app/Contents/MacOS/webstorm"},
	{name: "RustRover", bin: "rustrover", args: []string{"--wait"}, gui: true,
		macApp: "/Applications/RustRover.app/Contents/MacOS/rustrover"},
	{name: "Fleet", bin: "fleet", args: []string{"--wait"}, gui: true},
}

// Detect returns the editors present on this machine, terminal ones first.
//
// Terminal editors sort first because they keep you inside the terminal, which is the
// point of a TUI; a GUI editor is a deliberate choice rather than a default.
func Detect() []Editor {
	var found []Editor
	seen := map[string]bool{}

	for _, c := range known {
		path := resolve(c)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		found = append(found, Editor{
			Name: c.name, Command: path, Args: c.args, GUI: c.gui,
		})
	}

	sort.SliceStable(found, func(i, j int) bool {
		if found[i].GUI != found[j].GUI {
			return !found[i].GUI
		}
		return false // otherwise keep the declared order
	})
	return found
}

// resolve finds an editor's executable, checking PATH first and then the macOS bundle
// helper. Sublime and the JetBrains launchers are frequently installed without their
// CLI helper being linked into PATH.
func resolve(c candidate) string {
	if p, err := exec.LookPath(c.bin); err == nil {
		return p
	}
	if c.macApp != "" && runtime.GOOS == "darwin" {
		if fi, err := os.Stat(c.macApp); err == nil && !fi.IsDir() {
			return c.macApp
		}
	}
	return ""
}

// Lookup finds a detected editor by name or by command, so a configured value survives
// whichever form the user wrote.
func Lookup(nameOrCommand string) (Editor, bool) {
	if nameOrCommand == "" {
		return Editor{}, false
	}
	for _, e := range Detect() {
		if e.Name == nameOrCommand || e.Command == nameOrCommand ||
			filepath.Base(e.Command) == nameOrCommand {
			return e, true
		}
	}
	return Editor{}, false
}

// FromCommand builds an Editor for a command leetui does not know about, so a
// hand-written config value still works.
//
// Args are empty because there is no way to guess a blocking flag for an unknown
// program, and passing the wrong one is worse than passing none.
func FromCommand(cmd string) Editor {
	return Editor{Name: filepath.Base(cmd), Command: cmd}
}
