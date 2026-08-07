package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// The local run result.
//
// A failure that only says "1 of 3 failed" is not actionable — the whole reason to run
// locally is to see WHAT differed without a round-trip. So a failing case shows its
// input, what was expected, and what came out, stacked for comparison rather than
// side-by-side: terminal panes are narrow, and two 30-column columns truncate exactly
// the arrays you need to read.
//
// Passing cases collapse to one line. The failure is what needs the room.

// hasResult reports whether a fresh run belongs to the problem on screen.
//
// A result from another problem is worse than none — it invites reading the wrong
// failure — so it is shown only when the slugs agree.
func (m Model) hasResult() bool {
	return m.runResult != nil && m.runSlug != "" &&
		m.cursor < len(m.rows) && m.rows[m.cursor].Slug == m.runSlug
}

func (m Model) viewResult(w, h int) string {
	r := m.runResult
	passed, failed, errored := r.Summary()

	f := components.Frame{
		Title:   "run",
		Right:   resultTally(passed, failed, errored),
		Width:   w,
		Height:  h,
		Focused: m.focus == paneQueue,
	}

	var b strings.Builder
	inner := f.InnerWidth() - 2

	if r.CompileErr != "" {
		b.WriteString(" " + theme.Label.Render("Did not compile") + "\n\n")
		for _, line := range wrapPlain(r.CompileErr, inner) {
			b.WriteString(" " + theme.Body.Render(line) + "\n")
		}
		return f.Render(b.String())
	}

	for i, c := range r.Cases {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.viewCase(i, c, inner))
	}
	return f.Render(b.String())
}

// resultTally is the bezel label. It leads with whatever went wrong, because that is
// what the reader is looking for.
func resultTally(passed, failed, errored int) string {
	switch {
	case errored > 0:
		return fmt.Sprintf("%d crashed", errored)
	case failed > 0:
		return fmt.Sprintf("%d failed", failed)
	case passed > 0:
		return fmt.Sprintf("%d passed", passed)
	default:
		return "no verdict"
	}
}

func (m Model) viewCase(i int, c runner.CaseResult, inner int) string {
	var b strings.Builder
	label := theme.Meta.Render(fmt.Sprintf("case %d", i+1))

	switch {
	case c.Err != nil:
		b.WriteString(" " + label + "  " + theme.CaseFail.Render("CRASHED") + "\n")
		for _, line := range wrapPlain(c.Err.Error(), inner) {
			b.WriteString("   " + theme.Meta.Render(line) + "\n")
		}
		return b.String()

	case !c.Judged:
		// Neither passed nor failed, so neither colour. This case ran and there was
		// nothing to check it against — saying so in grey is the honest answer.
		b.WriteString(" " + label + "  " + theme.Meta.Render("NO EXPECTED ANSWER") + "\n")
		b.WriteString(field("out", c.Actual, inner))
		return b.String()

	case c.Passed:
		// A passing case collapses: it needs acknowledgement, not inspection.
		b.WriteString(" " + label + "  " + theme.CasePass.Render("PASS") +
			"  " + theme.Meta.Render(truncate(oneLine(c.Actual), inner-16)) + "\n")
		return b.String()

	default:
		b.WriteString(" " + label + "  " + theme.CaseFail.Render("FAIL") + "\n")
		b.WriteString(field("in ", c.Case.Input, inner))
		b.WriteString(field("want", c.Case.Expected, inner))
		b.WriteString(field("got ", c.Actual, inner))
		return b.String()
	}
}

// field renders one labelled value, wrapping rather than truncating: the differing tail
// of a long array is exactly the part worth seeing.
func field(label, value string, inner int) string {
	var b strings.Builder
	lines := wrapPlain(oneLine(value), inner-6)
	if len(lines) == 0 {
		lines = []string{""}
	}
	for i, line := range lines {
		tag := "    "
		if i == 0 {
			tag = " " + theme.Utility.Render(label)
		}
		b.WriteString(tag + " " + theme.Body.Render(line) + "\n")
	}
	return b.String()
}

// oneLine flattens multi-line input so a case stays scannable. Test inputs are one line
// per parameter; joined with " · " they read as an argument list.
func oneLine(s string) string {
	parts := strings.Split(strings.TrimSpace(s), "\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, theme.Rule.Render(" · "))
}

// wrapPlain hard-wraps at width, counting display cells so styled text does not overflow.
func wrapPlain(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	for _, para := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		for lipgloss.Width(para) > width {
			cut := width
			for cut > 0 && lipgloss.Width(para[:cut]) > width {
				cut--
			}
			out = append(out, para[:cut])
			para = para[cut:]
		}
		out = append(out, para)
	}
	return out
}
