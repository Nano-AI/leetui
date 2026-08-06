package main

import (
	"flag"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/tui"
	"github.com/Nano-AI/leetui/internal/tui/components"
)

// runTUI opens the app. This is what `leetui` with no subcommand does, and it is the
// product — the subcommands exist so editors can reach the same core (D-015).
func runTUI(args []string) error {
	fs := flag.NewFlagSet("leetui", flag.ContinueOnError)
	noMotion := fs.Bool("no-motion", false, "settle the flip animation instantly")
	if err := fs.Parse(args); err != nil {
		return err
	}

	a, err := open()
	if err != nil {
		return err
	}
	defer a.Close()

	// Motion is opt-out via flag or config. The app stays fully legible without it —
	// motion never carries information on its own.
	components.ReduceMotion = *noMotion || a.cfg.UI.ReduceMotion

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if a.cfg.UI.Mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(tui.New(a.cfg, a.store, a.client, a.sync), opts...)
	_, err = p.Run()
	return err
}
