package editor

import (
	"os"
	"os/exec"
)

// Splitting the terminal.
//
// A terminal editor and leetui both want the whole terminal, so handing it over means the
// statement disappears exactly when you start writing code against it. Inside a
// multiplexer there is a better answer: open the editor in a NEW PANE and leave leetui
// running in this one. The statement stays on screen, and the file watcher keeps working,
// so saving re-runs the tests without leaving the editor at all (D-012).
//
// Detection is by environment variable rather than by asking the binary. $TMUX is only set
// inside a session, so its presence is proof — whereas `tmux` being installed says nothing
// about whether this terminal is in one.

// Splitter opens a command in a new pane beside the current one.
type Splitter struct {
	// Name is shown to the user, e.g. "tmux".
	Name string

	// env is the variable whose presence means we are inside this multiplexer.
	env string
	// bin is the control binary.
	bin string
	// split builds the arguments that open dir and run argv in a new pane.
	split func(dir string, argv []string) []string
}

// splitters are tried in order. Each is recognised by an environment variable the
// multiplexer sets inside a session and nowhere else.
var splitters = []Splitter{
	{
		Name: "tmux", env: "TMUX", bin: "tmux",
		// -h is a horizontal SPLIT, which puts the new pane to the RIGHT — tmux names the
		// split line, not the arrangement. Focus follows into the editor, which is what
		// pressing `e` asks for; leetui stays visible in the pane beside it.
		split: func(dir string, argv []string) []string {
			return append([]string{"split-window", "-h", "-c", dir, "--"}, argv...)
		},
	},
	{
		Name: "zellij", env: "ZELLIJ", bin: "zellij",
		split: func(dir string, argv []string) []string {
			return append([]string{"run", "--direction", "right", "--cwd", dir, "--"}, argv...)
		},
	},
	{
		Name: "WezTerm", env: "WEZTERM_PANE", bin: "wezterm",
		split: func(dir string, argv []string) []string {
			return append([]string{"cli", "split-pane", "--right", "--cwd", dir, "--"}, argv...)
		},
	},
	{
		Name: "Kitty", env: "KITTY_WINDOW_ID", bin: "kitty",
		// Kitty needs allow_remote_control in its config. When it is off this fails
		// cleanly and the caller falls back to taking over the terminal.
		split: func(dir string, argv []string) []string {
			return append([]string{"@", "launch", "--type=window", "--cwd", dir}, argv...)
		},
	},
}

// DetectSplitter returns the multiplexer this terminal is running inside.
//
// Returns false when there is none, which is not a failure — it means the editor has to
// take over the terminal instead, and the caller has a working fallback for that.
func DetectSplitter() (Splitter, bool) {
	for _, s := range splitters {
		if os.Getenv(s.env) == "" {
			continue
		}
		path, err := exec.LookPath(s.bin)
		if err != nil {
			// Inside a session but the control binary is gone. Nothing to do but fall
			// back; reporting it would be noise about a terminal the user is using fine.
			continue
		}
		s.bin = path
		return s, true
	}
	return Splitter{}, false
}

// Command builds the invocation that opens argv in a new pane rooted at dir.
//
// It returns quickly whether or not the editor does: every multiplexer's control binary
// creates the pane and exits, so this is safe to run and wait on.
func (s Splitter) Command(dir string, argv []string) *exec.Cmd {
	// Arguments are passed as a slice, never a shell string — the directory is built
	// from a LeetCode slug.
	return exec.Command(s.bin, s.split(dir, argv)...)
}

// Available reports whether this splitter is usable.
func (s Splitter) Available() bool { return s.bin != "" }
