package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Type treatments
// ---------------------------------------------------------------------------
//
// The terminal offers one family, so the three "typefaces" are three treatments.
// They must stay visually distinct or the hierarchy collapses. Hierarchy comes from
// weight -> case -> color -> indentation, applied in that order. Never fake a larger
// size with box-drawing characters.

var (
	// Body is the content voice. Problem statements, editorial prose, code.
	// Never letterspaced, never uppercased.
	Body = lipgloss.NewStyle().Foreground(Bone)

	// Utility is column headers, eyebrows, and key hints: uppercase, dim, tight.
	Utility = lipgloss.NewStyle().Foreground(Dim)

	// Meta is inline secondary detail that sits beside body text.
	Meta = lipgloss.NewStyle().Foreground(Dim)

	// Label is the system speaking: pane titles, selection, the timer, the wordmark.
	Label = lipgloss.NewStyle().Foreground(Amber)

	// Rule draws every divider and pane border in the app.
	Rule = lipgloss.NewStyle().Foreground(Hinge)

	// RuleFocused marks the focused pane. Focus is indicated by the rule changing
	// color — never by a border style change or a background shift.
	RuleFocused = lipgloss.NewStyle().Foreground(Amber)
)

// Display renders the "flap face" treatment: uppercase, bold, one space between every
// character. Reserved for verdicts and the wordmark. Nothing else is ever letterspaced.
//
//	Display("accepted") => "A C C E P T E D"
//
// The letterspacing is load-bearing, not decorative: under NO_COLOR the verdict colors
// vanish, and this treatment is what still distinguishes a verdict from body text.
func Display(s string) string {
	return spaced(strings.ToUpper(s))
}

// UtilityText renders the utility treatment's text transform. Pair with the Utility
// style for color:
//
//	Utility.Render(UtilityText("submission queue")) => dim "SUBMISSION QUEUE"
func UtilityText(s string) string {
	return strings.ToUpper(s)
}

// spaced inserts a single space between runes, rune-safe.
func spaced(s string) string {
	r := []rune(s)
	if len(r) < 2 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s)*2 - 1)
	for i, c := range r {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(c)
	}
	return b.String()
}
