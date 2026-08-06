package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// judgeSummary phrases a verdict for the status line.
//
// It uses LeetCode's own wording for the verdict, then adds the one number that matters
// for that outcome: percentile when accepted, which case failed when not.
func judgeSummary(j leetcode.Judgement) string {
	switch {
	case j.Accepted():
		s := "Accepted"
		if j.Runtime != "" {
			s += fmt.Sprintf(" — %s", j.Runtime)
		}
		if j.RuntimePercentile > 0 {
			s += fmt.Sprintf(", beats %.1f%%", j.RuntimePercentile)
		}
		return s

	case j.StatusCode == leetcode.VerdictCompileError:
		msg := firstLine(j.CompileError, j.FullCompileError)
		if msg == "" {
			return "Compile error."
		}
		return "Compile error: " + msg

	case j.StatusCode == leetcode.VerdictRuntimeError:
		msg := firstLine(j.RuntimeError)
		if msg == "" {
			return "Runtime error."
		}
		return "Runtime error: " + msg

	case j.StatusCode == leetcode.VerdictWrongAnswer:
		s := "Wrong answer"
		if j.TotalTestcases > 0 {
			s += fmt.Sprintf(" — %d of %d cases passed", j.TotalCorrect, j.TotalTestcases)
		}
		if j.LastTestcase != "" {
			s += fmt.Sprintf(", failed on %s", strings.ReplaceAll(j.LastTestcase, "\n", " "))
		}
		return s

	default:
		if j.StatusMsg != "" {
			// LeetCode's own vocabulary, verbatim — translating it would make results
			// harder to reconcile against the website.
			return j.StatusMsg
		}
		return "The judge returned no verdict."
	}
}

func judgeErrorHint(err error) string {
	switch {
	case errors.Is(err, leetcode.ErrSessionExpired):
		return "Session expired. Press a to sign in again, then submit."
	case errors.Is(err, leetcode.ErrRateLimited):
		return "LeetCode is rate limiting. Wait a moment and submit again."
	default:
		return "Submit failed: " + err.Error()
	}
}

// firstLine returns the first non-empty line of the first non-empty candidate. A
// compiler's first line names the error; the rest is context that does not fit a status
// line.
func firstLine(candidates ...string) string {
	for _, c := range candidates {
		for _, line := range strings.Split(c, "\n") {
			if s := strings.TrimSpace(line); s != "" {
				return s
			}
		}
	}
	return ""
}
