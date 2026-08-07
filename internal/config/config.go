package config

// Config is the full user configuration. Every field has a working zero-value default
// applied by applyDefaults, so a missing or partial config file is never an error.
type Config struct {
	// Workspace is where problem folders are written (D-010).
	Workspace string `toml:"workspace"`

	// DefaultLang is the language new solution files are generated in.
	// Local execution is only available for go, python3, cpp, and rust (D-004);
	// anything else still edits and submits, it just runs on the judge.
	DefaultLang string `toml:"default_lang"`

	// LastLang is the language most recently used to create a solution file.
	//
	// Separate from DefaultLang, which is a preference the user sets. This is a habit the
	// app observes, and it wins at startup: whatever you were writing yesterday is a far
	// better guess than a default chosen once during setup. Empty until you create a file.
	LastLang string `toml:"last_lang"`

	// Editor overrides $EDITOR. Empty means use the environment (D-012).
	Editor string `toml:"editor"`

	// WatchSolution re-runs the tests when the solution file changes on disk, so an
	// editor open in another pane drives the loop without touching leetui (D-012).
	WatchSolution bool `toml:"watch_solution"`

	// EditorPane opens the editor in a NEW PANE when leetui is running inside tmux,
	// zellij, WezTerm, or Kitty, instead of taking over the terminal.
	//
	// This is what lets you read the statement while writing the solution: leetui stays
	// on screen, and the watcher keeps re-running on save. Off falls back to handing the
	// terminal over, which is also what happens outside a multiplexer.
	EditorPane bool `toml:"editor_pane"`

	// RunAfterEdit runs the tests once the editor exits, when the file actually changed.
	//
	// Only matters on the path where the editor took over the terminal — with a pane or
	// a GUI window the watcher has already been running them on every save.
	RunAfterEdit bool `toml:"run_after_edit"`

	// OpenStatement opens README.md alongside the solution, so the problem is readable
	// from inside an editor that has taken over the terminal. Editors with no known
	// split flag ignore it rather than opening the file stacked.
	OpenStatement bool `toml:"open_statement"`

	UI   UI   `toml:"ui"`
	Sync Sync `toml:"sync"`
	Git  Git  `toml:"git"`

	// Keys overrides individual bindings by action name, e.g. keys.submit = "S".
	// Actions not listed keep their default. See DefaultKeymap.
	Keys map[string]string `toml:"keys"`

	// path is where this config was loaded from; not serialized.
	path string
}

// UI holds presentation preferences.
type UI struct {
	// ReduceMotion settles the flip animation instantly. The app stays fully legible
	// without motion — the flip never carries information on its own.
	ReduceMotion bool `toml:"reduce_motion"`

	// ASCII replaces box-drawing and half-block glyphs with ASCII, for terminals with
	// poor font coverage.
	ASCII bool `toml:"ascii"`

	// InlineImages opts into the Kitty/iTerm2 graphics protocol for editorial images.
	// Off by default: the bracket marker ("[> img 1 — dp table]") is the floor and
	// works in every terminal (D-007).
	InlineImages bool `toml:"inline_images"`

	// Mouse enables click and scroll.
	Mouse bool `toml:"mouse"`
}

// Sync holds problem- and company-sync behaviour.
type Sync struct {
	// RequestsPerSecond caps ALL outbound LeetCode traffic. One limiter, no exceptions
	// (D-008). Raising this risks a rate-limit or an account flag.
	RequestsPerSecond float64 `toml:"requests_per_second"`

	// PageSize is how many problems are pulled per problemsetQuestionList call.
	PageSize int `toml:"page_size"`

	// CompanyRefreshDays is how stale company data may get before a refresh is offered.
	CompanyRefreshDays int `toml:"company_refresh_days"`
}

// Git holds repository behaviour (D-011).
type Git struct {
	// CommitOnAccepted auto-commits when the judge returns Accepted.
	CommitOnAccepted bool `toml:"commit_on_accepted"`

	// AutoPush is deliberately false by default and should stay that way: pushing is an
	// outward-facing action, and the app must never surprise-publish to a remote.
	AutoPush bool `toml:"auto_push"`

	// Branch is the branch commits are expected on. Empty means whatever is checked
	// out. When set, a mismatch aborts the commit rather than writing to the wrong place.
	Branch string `toml:"branch"`

	// Remote is the remote a push goes to. Empty means the branch's own upstream, and
	// a branch with no upstream is not pushed at all — leetui will not pick a remote
	// for solutions on the user's behalf.
	Remote string `toml:"remote"`

	// CommitNotes controls whether notes.md is committed alongside solutions.
	CommitNotes bool `toml:"commit_notes"`
}
