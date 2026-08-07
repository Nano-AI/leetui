package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/runner"
)

// runDoctor reports what this machine can and cannot do, and how to fix the gaps.
//
// It exists because every one of these failures is silent by default. A missing C++
// toolchain shows up as a run that will not start; tmux swallowing graphics shows up as a
// figure that never appears. Neither says which knob to turn, and the user reasonably
// concludes the app is broken.
//
// Read-only. It changes nothing and prints the exact command for anything it finds.
func runDoctor(a *app, args []string) (int, error) {
	fs, _ := flags("doctor")
	if _, err := parseFlags(fs, args); err != nil {
		return exitProblem, err
	}

	w := os.Stdout
	problems := 0

	fmt.Fprintln(w, "workspace")
	fmt.Fprintf(w, "  %-14s %s\n", "path", a.cfg.Workspace)
	if _, err := os.Stat(a.cfg.Workspace); err != nil {
		fmt.Fprintf(w, "  %-14s not created yet — it appears on the first pull\n", "state")
	}

	fmt.Fprintln(w, "\nlocal runners")
	engine := runner.NewLocal()
	for _, l := range runner.Langs() {
		if !l.Local {
			continue
		}
		if engine.Supports(l) {
			fmt.Fprintf(w, "  %-14s ok\n", l.Display)
			continue
		}
		problems++
		fmt.Fprintf(w, "  %-14s missing %s — install it, or submit instead\n",
			l.Display, engine.MissingToolchain(l))
	}

	fmt.Fprintln(w, "\neditor")
	if e, ok := editor.Lookup(a.cfg.Editor); ok {
		fmt.Fprintf(w, "  %-14s %s\n", "editor", e.Name)
	} else if found := editor.Detect(); len(found) > 0 {
		fmt.Fprintf(w, "  %-14s %s (detected)\n", "editor", found[0].Name)
	} else {
		problems++
		fmt.Fprintf(w, "  %-14s none found — set editor in %s\n", "editor", a.cfg.Path())
	}

	if s, ok := editor.DetectSplitter(); ok {
		fmt.Fprintf(w, "  %-14s %s — e opens a pane beside leetui\n", "splitter", s.Name)
	} else {
		fmt.Fprintf(w, "  %-14s none — e takes over the terminal\n", "splitter")
		fmt.Fprintf(w, "  %-14s run leetui inside tmux to keep the statement on screen\n", "")
	}

	problems += reportGraphics(w)

	// Where to look when something goes wrong, said before it does.
	fmt.Fprintln(w, "\ntracing")
	if a.LogPath != "" {
		fmt.Fprintf(w, "  %-14s on — %s\n", "debug log", a.LogPath)
	} else {
		fmt.Fprintf(w, "  %-14s off — leetui --debug, or LEETUI_DEBUG=1 for a subcommand\n",
			"debug log")
		fmt.Fprintf(w, "  %-14s credentials are redacted; it is safe to attach to a report\n", "")
	}

	fmt.Fprintln(w)
	if problems == 0 {
		fmt.Fprintln(w, "Nothing to fix.")
		return exitOK, nil
	}
	fmt.Fprintf(w, "%s to look at, listed above.\n", plural(problems, "thing"))
	return exitOK, nil
}

// reportGraphics says whether figures in a statement can be drawn, and what is stopping
// them when they cannot.
func reportGraphics(w io.Writer) int {
	g := config.DetectGraphics()

	fmt.Fprintln(w, "\ninline images")
	fmt.Fprintf(w, "  %-14s %s\n", "terminal", g.Terminal)

	// Figures currently open in a browser from the numbered markers in the statement.
	// Say that plainly wherever this lands: reporting "ok" for a feature that is not
	// built would be the same silent lie this command exists to prevent.
	if g.Protocol == config.ProtocolNone {
		fmt.Fprintf(w, "  %-14s no inline support — figures open in your browser, press 1-9\n",
			"images")
		return 0
	}
	fmt.Fprintf(w, "  %-14s %s\n", "protocol", g.Protocol)

	if !g.Blocked {
		fmt.Fprintf(w, "  %-14s ok — leetui image <problem> [n]\n", "images")
		fmt.Fprintf(w, "  %-14s inside the app, 1-9 opens a figure in your browser\n", "")
		return 0
	}

	// The whole reason this command exists. tmux drops the escape sequence and nothing
	// anywhere reports it, so a figure would simply never appear and no error is raised.
	fmt.Fprintf(w, "  %-14s BLOCKED by tmux — leetui image will draw nothing\n", "images")
	for _, line := range splitLines(g.Fix) {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintf(w, "  %-14s tmux must be RESTARTED after this, not just reloaded\n", "")
	return 1
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
