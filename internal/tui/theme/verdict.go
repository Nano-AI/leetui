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

// Resolved reports whether the judge has spoken. Unresolved verdicts render as a flap
// held mid-flip — that is the app's only loading state.
func (v Verdict) Resolved() bool { return v >= Accepted }

// Render returns the verdict in the display treatment, in its own color.
func (v Verdict) Render() string {
	return lipgloss.NewStyle().Foreground(v.Color()).Bold(true).Render(Display(v.Text()))
}
