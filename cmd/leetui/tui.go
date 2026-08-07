package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/tui"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// runTUI opens the app. This is what `leetui` with no subcommand does, and it is the
// product — the subcommands exist so editors can reach the same core (D-015).
func runTUI(args []string) error {
	fs := flag.NewFlagSet("leetui", flag.ContinueOnError)
	noMotion := fs.Bool("no-motion", false, "settle the flip animation instantly")
	ascii := fs.Bool("ascii", false, "draw with ASCII instead of box-drawing characters")
	debug := fs.Bool("debug", false, "write a redacted request trace to the debug log")
	if err := fs.Parse(args); err != nil {
		return err
	}

	a, err := open()
	if err != nil {
		return err
	}
	defer a.Close()

	if *debug {
		a.enableDebug()
	}
	// Said on stderr BEFORE the alternate screen takes over, or the one line telling the
	// user where to look scrolls past inside a full-screen app they cannot scroll.
	if a.LogPath != "" {
		fmt.Fprintf(os.Stderr, "leetui: tracing to %s\n", a.LogPath)
	}

	// Motion is opt-out via flag or config. The app stays fully legible without it —
	// motion never carries information on its own.
	components.ReduceMotion = *noMotion || a.cfg.UI.ReduceMotion

	// Decided once, from the flag, the config, and the environment. A terminal that
	// cannot draw the state column's glyphs at one cell each would shear the whole grid,
	// and the same terminal cannot draw the bezel either (D-026).
	//
	// The flag wins outright: it is the escape hatch for a terminal the detection reads
	// wrong, and someone who typed --ascii has already decided.
	theme.ASCII = *ascii || config.PreferASCII(a.cfg)

	// How much colour there is, asked once. Everything below truecolor still works —
	// nothing in this app is encoded in colour alone — but a monochrome terminal loses
	// the amber bezel that marks focus, so the panes grow a marker instead.
	theme.Active = theme.DetectProfile()

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if a.cfg.UI.Mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(tui.New(a.cfg, a.store, a.client, a.sync), opts...)
	_, err = p.Run()
	return err
}
