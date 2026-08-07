package config

import (
	"os"
	"os/exec"
	"strings"
)

// What this terminal can draw a picture with, and what is standing in the way.
//
// Statements are full of figures — a tree, a grid, a diagram of the array after each
// step — and a terminal that can show them inline should. Two protocols matter in
// practice: Kitty's, which WezTerm and Ghostty also speak, and iTerm2's.
//
// The awkward part is not detection, it is the multiplexer. Both protocols work by
// writing an escape sequence the terminal intercepts, and tmux eats anything it does not
// recognise unless it is told to pass it through. That option is OFF by default, so the
// common setup — kitty running tmux — silently drops every image with no error anywhere.
// Silently is the problem: the user concludes the feature is broken.
//
// So the check below reports not just what is possible but what is BLOCKED and how to
// unblock it, and the caller is expected to say so out loud.

// Protocol is an inline-image protocol a terminal understands.
type Protocol string

const (
	ProtocolNone  Protocol = ""
	ProtocolKitty Protocol = "kitty"
	ProtocolITerm Protocol = "iterm2"
)

// Graphics is what this terminal can do with an image, right now, as configured.
type Graphics struct {
	// Protocol the terminal speaks, ignoring whether anything is blocking it.
	Protocol Protocol

	// Terminal is the name to show the user, e.g. "kitty".
	Terminal string

	// InTmux reports whether this process is inside a tmux session.
	InTmux bool

	// Blocked is set when the terminal could draw images but the environment will not
	// let the escape sequence reach it.
	Blocked bool

	// Fix is the exact thing to change, ready to show. Empty when nothing is wrong.
	Fix string
}

// Available reports whether an image can actually be drawn as things stand.
func (g Graphics) Available() bool {
	return g.Protocol != ProtocolNone && !g.Blocked
}

// DetectGraphics works out what this terminal can show.
func DetectGraphics() Graphics {
	g := Graphics{InTmux: os.Getenv("TMUX") != ""}
	g.Protocol, g.Terminal = detectProtocol()

	if g.Protocol == ProtocolNone || !g.InTmux {
		return g
	}

	// Inside tmux the sequence only survives with passthrough on. Ask tmux itself
	// rather than reading a config file: the option can be set from anywhere, and the
	// running server is the only authority on its current value.
	if tmuxPassthrough() {
		return g
	}

	g.Blocked = true
	g.Fix = "tmux is not passing graphics through. Add this to ~/.tmux.conf:\n" +
		"    set -g allow-passthrough on\n" +
		"then reload with:  tmux source-file ~/.tmux.conf"
	return g
}

// detectProtocol identifies the terminal.
//
// Environment first, TERM second. A terminal sets its own variable inside its own
// windows and nowhere else, which is proof; TERM can be inherited or overridden by a
// remote shell, so it is the weaker signal and comes after.
func detectProtocol() (Protocol, string) {
	term := os.Getenv("TERM")
	prog := os.Getenv("TERM_PROGRAM")

	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return ProtocolKitty, "kitty"
	case os.Getenv("GHOSTTY_RESOURCES_DIR") != "", prog == "ghostty":
		return ProtocolKitty, "Ghostty"
	case os.Getenv("WEZTERM_PANE") != "":
		// WezTerm speaks both; Kitty's is the better specified of the two.
		return ProtocolKitty, "WezTerm"
	case prog == "iTerm.app":
		return ProtocolITerm, "iTerm2"
	case strings.Contains(term, "kitty"):
		return ProtocolKitty, "kitty"
	}
	return ProtocolNone, terminalName(term, prog)
}

func terminalName(term, prog string) string {
	if prog != "" {
		return prog
	}
	if term != "" {
		return term
	}
	return "this terminal"
}

// tmuxPassthrough asks the running tmux server whether it forwards unknown escapes.
//
// A missing binary or a failed call is treated as "not on": claiming images will work
// and then drawing nothing is worse than an unnecessary hint.
func tmuxPassthrough() bool {
	out, err := exec.Command("tmux", "show", "-gv", "allow-passthrough").Output()
	if err != nil {
		return false
	}
	v := strings.TrimSpace(string(out))
	return v == "on" || v == "all"
}
