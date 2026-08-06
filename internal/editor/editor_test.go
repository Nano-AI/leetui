package editor

import (
	"strings"
	"testing"
)

func TestDetectFindsSomething(t *testing.T) {
	found := Detect()
	for _, e := range found {
		t.Logf("%-20s %-8v %s %v", e.Name, e.GUI, e.Command, e.Args)
	}
	if len(found) == 0 {
		t.Skip("no known editor on this machine")
	}

	for _, e := range found {
		if e.Command == "" {
			t.Errorf("%s resolved to an empty command", e.Name)
		}
		// Every GUI editor must block, or the TUI redraws over an untouched file.
		if e.GUI && len(e.Args) == 0 {
			t.Errorf("%s is a GUI editor with no blocking flag", e.Name)
		}
	}
}

// TestTerminalEditorsSortFirst: a TUI should default to staying in the terminal.
func TestTerminalEditorsSortFirst(t *testing.T) {
	found := Detect()
	seenGUI := false
	for _, e := range found {
		if e.GUI {
			seenGUI = true
			continue
		}
		if seenGUI {
			t.Errorf("terminal editor %s sorted after a GUI one", e.Name)
		}
	}
}

func TestLaunchAppendsPath(t *testing.T) {
	e := Editor{Name: "VS Code", Command: "code", Args: []string{"--wait"}, GUI: true}
	cmd, args := e.Launch("/tmp/solution.py")
	if cmd != "code" {
		t.Errorf("command = %q", cmd)
	}
	if len(args) != 2 || args[0] != "--wait" || args[1] != "/tmp/solution.py" {
		t.Errorf("args = %v, want [--wait /tmp/solution.py]", args)
	}

	// Launch must not mutate the shared Args slice — a second call would otherwise
	// accumulate paths.
	_, again := e.Launch("/tmp/other.py")
	if len(again) != 2 || again[1] != "/tmp/other.py" {
		t.Errorf("second launch = %v; Args was mutated", again)
	}
}

func TestLookupAcceptsNameOrCommand(t *testing.T) {
	found := Detect()
	if len(found) == 0 {
		t.Skip("no editor to look up")
	}
	want := found[0]

	for _, key := range []string{want.Name, want.Command} {
		got, ok := Lookup(key)
		if !ok || got.Command != want.Command {
			t.Errorf("Lookup(%q) = %+v, %v", key, got, ok)
		}
	}
	if _, ok := Lookup("definitely-not-an-editor"); ok {
		t.Error("looked up an editor that does not exist")
	}
	if _, ok := Lookup(""); ok {
		t.Error("empty lookup returned an editor")
	}
}

// TestFromCommand: a hand-written config value must still work, with no guessed flags.
func TestFromCommand(t *testing.T) {
	e := FromCommand("/opt/homebrew/bin/nvim")
	if e.Name != "nvim" {
		t.Errorf("name = %q, want nvim", e.Name)
	}
	if len(e.Args) != 0 {
		t.Errorf("guessed args %v for an unknown editor", e.Args)
	}
	if _, args := e.Launch("/tmp/x.py"); len(args) != 1 || !strings.HasSuffix(args[0], "x.py") {
		t.Errorf("launch args = %v", args)
	}
}
