package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// clearMultiplexer makes the test independent of whether it is run inside tmux.
func clearMultiplexer(t *testing.T) {
	t.Helper()
	for _, v := range []string{"TMUX", "ZELLIJ", "WEZTERM_PANE", "KITTY_WINDOW_ID"} {
		t.Setenv(v, "")
	}
}

func problemDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, workspace.ReadmeFile),
		[]byte("# 1. Two Sum\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestGUIEditorNeverTakesTheTerminal is the fix for reading and writing at once with a
// windowed editor: blocking leetui behind a window that is already open buys nothing and
// costs the statement and the file watcher.
func TestGUIEditorNeverTakesTheTerminal(t *testing.T) {
	clearMultiplexer(t)
	dir := problemDir(t)
	code := editor.Editor{Name: "VS Code", Command: "code", Args: []string{"--wait"}, GUI: true}

	p := planEdit(config.Default(), code, dir, filepath.Join(dir, "solution.py"))

	if p.route != routeDetached {
		t.Fatalf("route = %v, want routeDetached", p.route)
	}
	// --wait is what makes ExecProcess correct. Detached, it would freeze leetui for as
	// long as the window is open, which is the whole problem.
	for _, a := range p.argv {
		if a == "--wait" {
			t.Errorf("detached launch kept --wait: %v", p.argv)
		}
	}
}

// TestTerminalEditorUsesAPaneWhenThereIsOne: the statement stays on screen and the
// watcher keeps working, so the tests re-run on save without switching back.
func TestTerminalEditorUsesAPaneWhenThereIsOne(t *testing.T) {
	clearMultiplexer(t)
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	t.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	nvim := editor.Editor{Name: "Neovim", Command: "nvim"}
	p := planEdit(config.Default(), nvim, problemDir(t), "solution.py")

	if p.route != routePane {
		t.Fatalf("route = %v, want routePane inside tmux", p.route)
	}
	if p.splitter.Name != "tmux" {
		t.Errorf("splitter is %q, want tmux", p.splitter.Name)
	}
	if !strings.Contains(p.note(), "tmux") {
		t.Errorf("the status line does not say where the editor went: %q", p.note())
	}
}

func TestEditorPaneCanBeTurnedOff(t *testing.T) {
	clearMultiplexer(t)
	t.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	cfg := config.Default()
	cfg.EditorPane = false
	p := planEdit(cfg, editor.Editor{Name: "Neovim", Command: "nvim"}, problemDir(t), "solution.py")

	if p.route != routeTakeover {
		t.Errorf("route = %v, want routeTakeover with editor_pane off", p.route)
	}
}

// TestTakeoverOpensTheStatement is the fallback answer to "how do I read the problem
// while working on it" when there is no pane to put leetui in.
func TestTakeoverOpensTheStatement(t *testing.T) {
	clearMultiplexer(t)
	dir := problemDir(t)

	p := planEdit(config.Default(), editor.Editor{Name: "Neovim", Command: "nvim"},
		dir, filepath.Join(dir, "solution.py"))

	if p.route != routeTakeover {
		t.Fatalf("route = %v, want routeTakeover", p.route)
	}
	joined := strings.Join(p.argv, " ")
	if !strings.Contains(joined, workspace.ReadmeFile) {
		t.Errorf("the statement was not opened alongside: %v", p.argv)
	}
	// The split flag has to come before the filenames or vim treats it as one.
	if p.argv[1] != "-O" {
		t.Errorf("split flag is not first: %v", p.argv)
	}
}

// TestUnknownEditorGetsNoSecondFile: an editor whose split flag we do not know would
// open README stacked over the solution, which is a buffer in the way rather than a
// statement to read.
func TestUnknownEditorGetsNoSecondFile(t *testing.T) {
	clearMultiplexer(t)
	dir := problemDir(t)

	p := planEdit(config.Default(), editor.Editor{Name: "Nano", Command: "nano"},
		dir, filepath.Join(dir, "solution.py"))

	if strings.Contains(strings.Join(p.argv, " "), workspace.ReadmeFile) {
		t.Errorf("nano was given a second file it cannot split: %v", p.argv)
	}
}

func TestOpenStatementCanBeTurnedOff(t *testing.T) {
	clearMultiplexer(t)
	dir := problemDir(t)

	cfg := config.Default()
	cfg.OpenStatement = false
	p := planEdit(cfg, editor.Editor{Name: "Neovim", Command: "nvim"},
		dir, filepath.Join(dir, "solution.py"))

	if strings.Contains(strings.Join(p.argv, " "), workspace.ReadmeFile) {
		t.Errorf("open_statement = false still opened the README: %v", p.argv)
	}
}
