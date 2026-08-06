package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
)

// app is everything a command needs, wired once.
//
// The TUI and the subcommands are two frontends over the same core, so they share this
// rather than each building their own — a second wiring path is how the two would end up
// reading different databases or ignoring the rate limiter.
type app struct {
	cfg    config.Config
	store  *store.Store
	client *leetcode.Client
	sync   *syncer.Syncer
}

// open wires the app. Callers must Close it.
func open() (*app, error) {
	cfg, err := config.Load()
	if err != nil {
		// A malformed config is worth telling the user about, but it must not stop the
		// app: Load already fell back to defaults.
		fmt.Fprintf(os.Stderr, "leetui: %v (using defaults)\n", err)
	}
	for _, p := range cfg.Validate() {
		fmt.Fprintf(os.Stderr, "leetui: config: %s\n", p)
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	st, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}

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
	return &app{
		cfg:    cfg,
		store:  st,
		client: client,
		sync:   syncer.New(client, st, cfg.Sync.PageSize),
	}, nil
}

func (a *app) Close() error { return a.store.Close() }

// problem resolves an argument to a problem, fetching its statement if it is not cached.
//
// The fetch is the difference between the CLI and the TUI: the TUI has already loaded the
// problem into a pane by the time you act on it, whereas `leetui run two-sum` may be the
// first time this machine has ever seen it.
func (a *app) problem(ctx context.Context, arg string) (*store.Detail, error) {
	slug, err := solve.Locate(arg)
	if err != nil {
		return nil, err
	}
	d, err := a.sync.Detail(ctx, slug, false)
	if err != nil {
		if errors.Is(err, leetcode.ErrNotFound) {
			return nil, fmt.Errorf("no problem called %q", slug)
		}
		return nil, err
	}
	return d, nil
}

// language picks the language for a command: the flag if given, otherwise the config
// default, otherwise whatever the solution file on disk already is.
//
// The last one matters for `leetui run %` from an editor: the file being edited names the
// language more reliably than a config default set months ago.
func (a *app) language(flagLang, path string) (runner.Lang, error) {
	if flagLang != "" {
		l, ok := runner.Lookup(flagLang)
		if !ok {
			return runner.Lang{}, fmt.Errorf("unknown language %q", flagLang)
		}
		return l, nil
	}
	if path != "" {
		if l, ok := runner.ByFilename(path); ok {
			return l, nil
		}
	}
	if l, ok := runner.Lookup(a.cfg.DefaultLang); ok {
		return l, nil
	}
	l, _ := runner.Lookup("python3")
	return l, nil
}
