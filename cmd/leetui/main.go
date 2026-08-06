// Command leetui is a terminal client for LeetCode.
//
// See docs/DECISIONS.md for the architecture record, docs/DESIGN.md for the visual
// system, and docs/ROADMAP.md for what is built so far.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/grootbeat/leetui/internal/auth"
	"github.com/grootbeat/leetui/internal/config"
	"github.com/grootbeat/leetui/internal/leetcode"
	"github.com/grootbeat/leetui/internal/store"
	"github.com/grootbeat/leetui/internal/syncer"
	"github.com/grootbeat/leetui/internal/tui"
	"github.com/grootbeat/leetui/internal/tui/components"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "leetui: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	noMotion := flag.Bool("no-motion", false, "settle the flip animation instantly")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		// A malformed config is worth telling the user about, but it must not stop the
		// app: Load already fell back to defaults.
		fmt.Fprintf(os.Stderr, "leetui: %v (using defaults)\n", err)
	}
	for _, p := range cfg.Validate() {
		fmt.Fprintf(os.Stderr, "leetui: config: %s\n", p)
	}

	// Motion is opt-out via flag or config. The app stays fully legible without it —
	// motion never carries information on its own.
	components.ReduceMotion = *noMotion || cfg.UI.ReduceMotion

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}
	st, err := store.Open(dataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	// Missing credentials are the ordinary first-run case, not a failure: the problem
	// list is public, so the app is useful before signing in.
	creds, err := auth.Load()
	if err != nil && !errors.Is(err, auth.ErrNoCredentials) {
		fmt.Fprintf(os.Stderr, "leetui: %v\n", err)
	}

	client := leetcode.New(
		leetcode.WithCredentials(creds),
		leetcode.WithRateLimit(cfg.Sync.RequestsPerSecond),
	)
	sy := syncer.New(client, st, cfg.Sync.PageSize)

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.UI.Mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(tui.New(cfg, st, client, sy), opts...)
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
