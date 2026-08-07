package theme

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Verdicts
// ---------------------------------------------------------------------------

// Verdict is a judge outcome. The zero value is Pending.
type Verdict int

const (
	Pending Verdict = iota
	Judging
	Accepted
	WrongAnswer
	TimeLimitExceeded
	MemoryLimitExceeded
	RuntimeError
	CompileError
)

// Text returns LeetCode's own wording for the verdict.
//
// The domain's real vocabulary is used verbatim and unabbreviated. Translating these
// into friendlier words would make results harder to reconcile against the website.
func (v Verdict) Text() string {
	switch v {
	case Judging:
		return "judging"
	case Accepted:
		return "accepted"
	case WrongAnswer:
		return "wrong answer"
	case TimeLimitExceeded:
		return "time limit exceeded"
	case MemoryLimitExceeded:
		return "memory limit exceeded"
	case RuntimeError:
		return "runtime error"
	case CompileError:
		return "compile error"
	default:
		return "pending"
	}
}

// Color returns the verdict's token. Unresolved states are amber — the system is still
// speaking, because the judge has not.
func (v Verdict) Color() lipgloss.Color {
	switch v {
	case Accepted:
		return AC
	case WrongAnswer, RuntimeError, CompileError:
		return WA
	case TimeLimitExceeded, MemoryLimitExceeded:
		return TLE
	default:
		return Amber
	}
}

// CasePass and CaseFail colour a LOCAL test case.
//
// Green and red were the judge's alone, and the reason was good: the verdict flip is the
// boldest moment in the app and nothing should compete with it. In practice the run panel
// is what you stare at while solving, and a column of amber FAILs beside a bone PASS is
// slower to scan than the two colours everyone already reads without thinking.
//
// So the distinction moved from COLOUR to TREATMENT. A judge's verdict is letterspaced —
// `A C C E P T E D`, the Display face, reserved for the judge and the wordmark. A local
// case is plain uppercase. Same palette, different voice, and the flip still lands.
//
// Bold rather than tinted-and-limp: these have to read at a glance from across the room,
// which is the only reason the column exists.
var (
	CasePass = lipgloss.NewStyle().Foreground(AC).Bold(true)
	CaseFail = lipgloss.NewStyle().Foreground(WA).Bold(true)
)

// Resolved reports whether the judge has spoken. Unresolved verdicts render as a flap
// held mid-flip — that is the app's only loading state.
func (v Verdict) Resolved() bool { return v >= Accepted }

// Render returns the verdict in the display treatment, in its own color.
func (v Verdict) Render() string {
	return lipgloss.NewStyle().Foreground(v.Color()).Bold(true).Render(Display(v.Text()))
}
