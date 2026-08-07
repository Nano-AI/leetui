package tui

// ---------------------------------------------------------------------------
// Focus and modes
// ---------------------------------------------------------------------------

// pane identifies the focused region. Focus is shown by that pane's rule turning amber —
// never by a border change or background shift (docs/DESIGN.md § Structure).
type pane int

const (
	paneBoard pane = iota
	paneDetail
	paneQueue

	paneCount
)

func (p pane) title() string {
	switch p {
	case paneDetail:
		return "detail"
	case paneQueue:
		return "submission queue"
	default:
		return "problems"
	}
}

// mode is the top-level screen. Overlays (help, auth, search) take the whole frame or
// a strip of it rather than being separate programs.
type mode int

const (
	modeBoard mode = iota
	modeSetup
	modeAuth
	modeHelp

	// modeCompany is the premium company browser (D-006). It is a mode rather than a
	// picker because it owns a text field: 984 companies need typing to narrow, and the
	// pickers are pure selection lists.
	modeCompany

	// modeSolve is one problem: statement, editorial, run results, submissions.
	//
	// Separate from the board because browsing and solving want opposite things from the
	// screen — a list wants width, a statement wants to be the only thing on it (D-018).
	modeSolve

	// modeGit is the workspace repository: branch, what is uncommitted, what is
	// unpushed, and the last few commits.
	//
	// A mode rather than a strip on the board. Pushing lives in here and nowhere else,
	// so publishing takes two deliberate keypresses and the second one is pressed while
	// looking at the remote's URL (D-011).
	modeGit
)

// Layout breakpoints. Panes collapse rather than truncating mid-glyph.
const (
	wideMin   = 100 // below this the queue folds away
	narrowMin = 80  // below this board and detail become full-screen, switched by tab
)
