package theme

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Tokens
// ---------------------------------------------------------------------------

// System tokens. Six, and no more. Each has exactly one job.
//
// The palette commits to a dark board rather than adapting to terminal background:
// a split-flap board is a lit object in a dark housing, and an adaptive light variant
// would be a different design wearing the same name.
const (
	Ink   = lipgloss.Color("#0F1014") // page base — the board's shadow box
	Flap  = lipgloss.Color("#1B1D24") // flap face — panel and row backgrounds
	Hinge = lipgloss.Color("#2A2D36") // the seam between flaps — every rule and divider
	Amber = lipgloss.Color("#E8A33D") // the system's voice — labels, selection, timer
	Bone  = lipgloss.Color("#E6E3DA") // content — problem text, code, prose
	Dim   = lipgloss.Color("#6B7080") // secondary — metadata, disabled, hints
)

// Verdict tokens. Reserved for the judge. Do not use these for anything else.
const (
	AC  = lipgloss.Color("#4FB477") // Accepted
	WA  = lipgloss.Color("#D65A5A") // Wrong Answer, Runtime Error, Compile Error
	TLE = lipgloss.Color("#C9822E") // Time Limit Exceeded, Memory Limit Exceeded
)
